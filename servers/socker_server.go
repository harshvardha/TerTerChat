package servers

import (
	"log"
	"net"
	"os"
	"sync"
)

func handleConnections(connection net.Conn) {
	defer connection.Close()
	log.Printf("TCP: New connection from: %s", connection.RemoteAddr())

	// creating a buffer to read and write from and to connection
	buffer := make([]byte, 1024)
	for {
		// reading the command
		n, err := connection.Read(buffer)
		if err != nil {
			log.Printf("TCP: Error reading from %s: %v", connection.RemoteAddr(), err)
			break
		}

		// check which event to handler using event handler
		event := string(buffer[:n])
		log.Printf("TCP: Recieved event")
	}
}

func StartTCPServer(port string, quit <-chan os.Signal, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Printf("Starting TCP server at port: %s", port)

	// creating a listner to listen connection infinetly until shutdown
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatal("TCP: Failed to start server: ", err)
	}
	defer listener.Close()

	go func() {
		<-quit
		log.Println("TCP: Shutdown signal recieved.")
		listener.Close()
	}()

	// listening for connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			// checking if error is due to listener being closed
			if opErr, ok := err.(*net.OpError); ok && opErr.Op == "accept" {
				log.Println("TCP: Listener closed, stopping connection accept loop.")
				break
			}
			log.Printf("TCP: Error accepting connections: %v", err)
			continue
		}

		// launching a go routine for handling each connection
		go handleConnections(conn)
	}

	log.Println("TCP: Socket server stopped.")
}
