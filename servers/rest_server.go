package servers

import (
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/harshvardha/TerTerChat/utility"
)

func StartRESTApiServer(port string, quit <-chan os.Signal, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Printf("REST: Starting server.")

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

	if err := server.ListenAndServe(); err != nil && err == http.ErrServerClosed {
		log.Fatalf("REST: server failed %v", err)
	}
	log.Println("REST: Server stopped.")
}
