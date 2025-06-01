package servers

import (
	"context"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/harshvardha/TerTerChat/controllers"
	"github.com/harshvardha/TerTerChat/utility"
)

func StartRESTApiServer(port string, apiConfig *controllers.ApiConfig, quit <-chan os.Signal, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Printf("[REST SERVER]: Starting server, [TIME]: %s", time.Now().Format(time.RFC1123))

	// define all the handlers here
	router := http.NewServeMux()
	router.HandleFunc("GET /api/v1/healthz", func(w http.ResponseWriter, r *http.Request) {
		utility.RespondWithJson(w, http.StatusOK, "OK")
	})

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  time.Second * 5,
		WriteTimeout: time.Second * 10,
		IdleTimeout:  time.Second * 120,
	}

	// creating a server error channel to shutdown server if an unexpected error occurs during ListenAndServer()
	serverErr := make(chan error, 1)

	// launching ListenAndServe in a separate go-routine
	go func() {
		if err := server.ListenAndServe(); err != nil && err == http.ErrServerClosed {
			log.Fatalf("[REST SERVER]: server failed %v, [TIME]: %s", err, time.Now().Format(time.RFC1123))
			serverErr <- err
		}
	}()

	// waiting for quit signal or serverErr channel to throw ErrServerClosed
	select {
	case sig := <-quit:
		// recieved a signal(for e.g., ctrl+c or SIGTERM)
		log.Printf("[REST SERVER]: signal recieved %s. shutting down serve, [TIME]: %s", sig, time.Now().Format(time.RFC1123))

		// initiating proper http server shutdown
		// creating a context with a timeout of 30 sec
		// which makes sure that shutdown process exits after 30 sec
		context, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel() // releasing all the resources acquired by the context when it done

		if err := server.Shutdown(context); err != nil {
			log.Fatalf("[REST SERVER]: server shutdown failed %v, [TIME]: %s", err, time.Now().Format(time.RFC1123))
		}

		log.Printf("[REST SERVER]: server shutdown successfull, [TIME]: %s", time.Now().Format(time.RFC1123))
	case err := <-serverErr:
		log.Fatalf("[REST SERVER]: server failed to start or crashed %v, [TIME]: %s", err, time.Now().Format(time.RFC1123))
	}
}
