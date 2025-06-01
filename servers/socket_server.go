package servers

import (
	"bufio"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	pingMessage  = "_PING_\n"
	pongMessage  = "_PONG_\n"
	pingInterval = 10 * time.Second
	pingTimeout  = 5 * time.Second
)

// heartbeat mechanism to check whether the client connection is still alive or not
func handleConnections(connection net.Conn, wg *sync.WaitGroup) {
	defer wg.Done()
	defer connection.Close()

	log.Printf("[CONNECTION ACCEPTED]: %s", connection.RemoteAddr())

	// creating channels for communication between readFromConnection and writeToConnection
	writer := make(chan []byte, 10)
	stopChan := make(chan struct{}) // this will be used to know whether to stop the heartbeat mechanism for the connection or not

	// reader go-routine
	go readFromConnection(connection, writer, stopChan)

	// writer go-routine
	go writeToConnection(connection, writer, stopChan)

	// blocking until signal recieved to stop heartbeat mechanism
	<-stopChan
	log.Printf("[CONNECTION CLOSED]: %s", connection.RemoteAddr())
}

// reader go-routine to read pong messages if server sends ping or respond with pong messages if client sends ping
func readFromConnection(connection net.Conn, writer chan<- []byte, stopChan chan<- struct{}) {
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
				log.Printf("[CONNECTION READER FOR %s]: connection timedout. connection dead", connection.RemoteAddr())
			} else if err == io.EOF {
				log.Printf("[CONNECTION READER FOR %s]: client connection closed.", connection.RemoteAddr())
			} else {
				log.Printf("[CONNECTION READER FOR %s]: error reading message from client, ERR: %v", connection.RemoteAddr(), err)
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

func writeToConnection(connection net.Conn, writer <-chan []byte, stopChan chan<- struct{}) {
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
				return //connection is broken
			}
		case <-ticker.C:
			connection.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if _, err := connection.Write([]byte(pingMessage)); err != nil {
				log.Printf("[CONNECTION WRITER FOR %s]: error writing to connection, ERR: %v", connection.RemoteAddr(), err)
				return // connection is broken
			}
		}
	}
}

func StartTCPServer(port string, quit <-chan os.Signal, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Printf("[TCP SERVER]: Starting TCP server at port: %s", port)

	// creating a listner to listen connection infinetly until shutdown
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatal("[TCP LISTENER]: Failed to start server: ", err)
	}
	defer listener.Close()

	go func() {
		<-quit
		log.Println("[TCP SERVER]: Shutdown signal recieved.")
		listener.Close()
	}()

	// listening for connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			// checking if error is due to listener being closed
			if opErr, ok := err.(*net.OpError); ok && opErr.Op == "accept" {
				log.Println("[TCP LISTENER]: Listener closed, stopping connection accept loop.")
				break
			}
			log.Printf("[TCP LISTENER]: Error accepting connections: %v", err)
			continue
		}

		// launching a go routine for handling each connection
		wg.Add(1)
		go handleConnections(conn, wg)
	}

	log.Println("[TCP SERVER]: Socket server stopped.")
}
