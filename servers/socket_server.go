package servers

import (
	"bufio"
	"crypto/tls"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	eventhandlers "github.com/harshvardha/TerTerChat/event_handlers"
	"github.com/harshvardha/TerTerChat/internal/database"
	"github.com/harshvardha/TerTerChat/internal/services"
)

const (
	pingMessage  = "_PING_\n"
	pongMessage  = "_PONG_\n"
	pingInterval = 10 * time.Second
	pingTimeout  = 5 * time.Second

	// server certificate and key file paths
	certificateFile = "server.crt"
	keyFile         = "server.key"
)

// heartbeat mechanism to check whether the client connection is still alive or not
func handleTLSConnections(connection net.Conn, phonenumber string, db *database.Queries, notificationService *services.Notification, connectionEventChannel chan eventhandlers.ConnectionEvent, wg *sync.WaitGroup) {
	defer wg.Done()
	defer connection.Close()

	log.Printf("[CONNECTION ACCEPTED]: %s", connection.RemoteAddr())

	// setting tcp keepalive for duration
	tlsConnection, ok := connection.(*tls.Conn)
	if !ok {
		log.Fatalf("[TCP SERVER]: not a tls connection")
	}

	// checking for handshake mechanism
	if err := tlsConnection.Handshake(); err != nil {
		log.Fatalf("[TCP SERVER]: unable to perform tls handshake: %v", err)
	}

	// accessing underlying raw tcp connection for setting up keepalive duration
	underlyingConnection := tlsConnection.NetConn()
	tcpConnection, ok := underlyingConnection.(*net.TCPConn)
	if !ok {
		log.Fatalf("[TCP SERVER]: underlying connection is not tcp")
	}

	// configuring tcp keepalive duration
	if err := tcpConnection.SetKeepAliveConfig(net.KeepAliveConfig{
		Enable:   true,
		Idle:     60 * time.Second,
		Interval: 10 * time.Second,
		Count:    5,
	},
	); err != nil {
		log.Fatalf("[TCP SERVER]: unable to set keepalive config for tcp connection: %v", err)
	}

	// creating channels for communication between readFromConnection and writeToConnection
	writer := make(chan []byte, 10)
	stopChan := make(chan struct{}) // this will be used to know whether to stop the heartbeat mechanism for the connection or not

	// reader go-routine
	go readFromConnection(connection, phonenumber, db, notificationService, connectionEventChannel, writer, stopChan)

	// writer go-routine
	go writeToConnection(connection, phonenumber, db, notificationService, connectionEventChannel, writer, stopChan)

	// blocking until signal recieved to stop heartbeat mechanism
	<-stopChan
	log.Printf("[CONNECTION CLOSED]: %s", connection.RemoteAddr())
}

// reader go-routine to read pong messages if server sends ping or respond with pong messages if client sends ping
func readFromConnection(connection net.Conn, phonenumber string, db *database.Queries, notificationService *services.Notification, connectionEventChannel chan eventhandlers.ConnectionEvent, writer chan<- []byte, stopChan chan<- struct{}) {
	defer func() {
		log.Printf("[CONNECTION READER FOR %s]: Exiiting", connection.RemoteAddr())
		stopChan <- struct{}{} //signal to stop the heartbeat mechanism
	}()

	reader := bufio.NewReader(connection)

	// reading from connection
	for {
		// setting read deadline on the connection after that it will be considered dead
		connection.SetReadDeadline(time.Now().Add(pingTimeout))
		message, err := reader.ReadBytes('\n')
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				log.Printf("[CONNECTION READER FOR %s]: connection timedout.", connection.RemoteAddr())
				continue
			} else if err == io.EOF {
				log.Printf("[CONNECTION READER FOR %s]: client connection closed.", connection.RemoteAddr())
			} else {
				log.Printf("[CONNECTION READER FOR %s]: error reading message from client, ERR: %v", connection.RemoteAddr(), err)
			}

			// emitting disconnect event to connection event handler
			connectionEventChannel <- eventhandlers.ConnectionEvent{
				Name:                "DISCONNECTED",
				Phonenumber:         phonenumber,
				ConnectionInstance:  nil,
				NotificationService: notificationService,
				DB:                  db,
				EmittedAt:           time.Now(),
			}

			return
		}

		// reading message
		msgString := strings.TrimSpace(string(message))
		switch msgString {
		case pingMessage:
			log.Printf("[CONNECTION READER FOR %s]: PING  message recieved. sending PONG", connection.RemoteAddr())

			// checking if the writer is available or not
			select {
			case writer <- []byte(pongMessage): // sending pong message to client
			default:
				log.Printf("[CONNECTION READER FOR %s]: writer channel busy. cannot send pong message", connection.RemoteAddr())
			}
		case pongMessage:
			log.Printf("[CONNECTION READER FOR %s]: recieved pong message.", connection.RemoteAddr())
		}
	}
}

func writeToConnection(connection net.Conn, phonenumber string, db *database.Queries, notificationService *services.Notification, connectionEventChannel chan eventhandlers.ConnectionEvent, writer <-chan []byte, stopChan chan<- struct{}) {
	ticker := time.NewTicker(10 * time.Second)
	defer func() {
		log.Printf("[CONNECTION WRITER FOR %s]: Exiting.", connection.RemoteAddr())
		ticker.Stop()
		stopChan <- struct{}{}
	}()

	for {
		select {
		case message, ok := <-writer:
			if !ok {
				log.Printf("[CONNECTION WRITER FOR %s]: writer channel closed.", connection.RemoteAddr())
				return // channel closed, writer will exit
			}
			connection.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if _, err := connection.Write(message); err != nil {
				log.Printf("[CONNECTION WRITER FOR %s]: error writing to connection, ERR: %v", connection.RemoteAddr(), err)

				// emitting connection disconnected event
				connectionEventChannel <- eventhandlers.ConnectionEvent{
					Name:                "DISCONNECTED",
					Phonenumber:         phonenumber,
					ConnectionInstance:  nil,
					DB:                  db,
					NotificationService: notificationService,
					EmittedAt:           time.Now(),
				}

				return
			}
		case <-ticker.C:
			connection.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if _, err := connection.Write([]byte(pingMessage)); err != nil {
				log.Printf("[CONNECTION WRITER FOR %s]: error writing to connection, ERR: %v", connection.RemoteAddr(), err)

				// emitting connection disconnected event
				connectionEventChannel <- eventhandlers.ConnectionEvent{
					Name:                "DISCONNECTED",
					Phonenumber:         phonenumber,
					ConnectionInstance:  nil,
					DB:                  db,
					NotificationService: notificationService,
					EmittedAt:           time.Now(),
				}

				return
			}
		}
	}
}

func StartTCPServer(port string, notificationService *services.Notification, db *database.Queries, connectionEventChannel chan eventhandlers.ConnectionEvent, quit <-chan os.Signal, wg *sync.WaitGroup) {
	defer wg.Done()

	// loading server certificate and private key
	certificate, err := tls.LoadX509KeyPair(certificateFile, keyFile)
	if err != nil {
		log.Fatalf("[TCP SERVER]: failed to load server certificate and key file: %v", err)
	}

	// configuring TLS for server
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{certificate},
		MinVersion:   tls.VersionTLS12,
		MaxVersion:   tls.VersionTLS13,
		CipherSuites: []uint16{tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256},
	}

	log.Printf("[TCP SERVER]: Starting TCP server at port: %s", port)

	// creating a listner to listen connection infinetly until shutdown
	listener, err := tls.Listen("tcp", ":"+port, tlsConfig)
	if err != nil {
		log.Fatal("[TCP LISTENER]: Failed to start server: ", err)
	}
	defer listener.Close()
	log.Printf("[TCP SERVER]: server listening securely on %s", port)

	go func() {
		<-quit
		log.Println("[TCP SERVER]: Shutdown signal recieved.")
		listener.Close()
	}()

	// creating a reader to read from connections when user get connected to add to notfication service
	buffer := make([]byte, 8)

	// listening for connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			// checking if error is due to listener being closed
			if opErr, ok := err.(*net.OpError); ok && opErr.Op == "accept" {
				log.Println("[TCP LISTENER]: Listener closed, stopping connection accept loop.")
				break
			}
			continue
		}

		// emitting connected event to connection event handler
		err = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		if err != nil {
			log.Printf("[TCP_SERVER]: error setting read deadline: %v", err)
		}
		n, err := conn.Read(buffer)
		if err != nil {
			log.Printf("[TCP SERVER]: error reading phonenumber from connection: %v", err)
			continue
		}

		phonenumber := strings.TrimSpace(string(buffer[:n]))
		connectionEventChannel <- eventhandlers.ConnectionEvent{
			Name:                "CONNECTED",
			Phonenumber:         phonenumber,
			ConnectionInstance:  conn,
			NotificationService: notificationService,
			DB:                  nil,
			EmittedAt:           time.Now(),
		}

		// launching a go routine for handling each connection
		wg.Add(1)
		go handleTLSConnections(conn, phonenumber, db, notificationService, connectionEventChannel, wg)
	}

	log.Println("[TCP SERVER]: Socket server stopped.")
}
