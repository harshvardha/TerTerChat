package main

import (
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/harshvardha/TerTerChat/servers"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	// loading tcp port env variable
	tcpPort := os.Getenv("TCP_PORT")
	if tcpPort == "" {
		log.Fatal("TCP port not set")
	}

	// loading rest api port env variable
	restApiPort := os.Getenv("REST_API_PORT")
	if restApiPort == "" {
		log.Fatal("REST api port not set")
	}

	var wg sync.WaitGroup

	// creating a quit channel to listen for os signal for shutting down servers
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// starting tcp server
	wg.Add(1)
	go servers.StartTCPServer(tcpPort, quit, &wg)

	// starting rest api server
	wg.Add(1)
	go servers.StartRESTApiServer(restApiPort, quit, &wg)

	// waiting for servers to shutdown
	<-quit
	log.Println("Shutting down servers...")
	wg.Wait()
}
