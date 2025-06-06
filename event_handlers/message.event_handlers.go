package eventhandlers

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/harshvardha/TerTerChat/internal/database"
	"github.com/harshvardha/TerTerChat/internal/services"
)

type MessageEvent struct {
	Name                string
	Phonenumbers        []string
	Message             database.Message
	NotificationService *services.Notification
	EmittedAt           time.Time
}

type message struct {
	Message     database.Message `json:"message,omitempty"`
	MessageType string           `json:"messageType"`
}

const (
	NEW_MESSAGE    = "NEW_MESSAGE"
	EDIT_MESSAGE   = "EDIT_MESSAGE"
	DELETE_MESSAGE = "DELETE_MESSAGE"
)

func MessageEventHandler(event chan MessageEvent, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Printf("[EVENT_HANDLER]: message event handler started")
	for messageEvent := range event {
		log.Printf("[EVENT]: %s, [TIME]: %s", messageEvent.Name, messageEvent.EmittedAt.Format(time.RFC1123))
		msg, err := json.Marshal(message{
			Message:     messageEvent.Message,
			MessageType: messageEvent.Name,
		})
		if err != nil {
			log.Printf("[EVENT]: Unable to marshal message: %v", err)
			return
		}

		go messageEvent.NotificationService.PushNotification(messageEvent.Phonenumbers, msg)
	}

	log.Printf("[EVENT]: Message event handler for %v stopped because event channel was closed", (<-event).Phonenumbers)
}
