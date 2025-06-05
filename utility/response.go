package utility

import (
	"encoding/json"
	"log"
	"net/http"
)

func RespondWithError(w http.ResponseWriter, code int, message string) {
	if code > 499 {
		log.Println("Responding with 5XX error: ", message)
	}

	type errorResponse struct {
		Error string `json:"error"`
	}

	RespondWithJson(w, code, errorResponse{
		Error: message,
	})
}

func RespondWithJson(w http.ResponseWriter, code int, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Println("Error marshaling payload to json")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if _, err = w.Write(data); err != nil {
		log.Printf("[REST_SERVER]: error writing response to connection: %v", err)
	}
}
