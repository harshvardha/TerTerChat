package eventhandlers

import (
	"encoding/json"
	"log"
	"time"

	"github.com/harshvardha/TerTerChat/internal/database"
	"github.com/harshvardha/TerTerChat/internal/services"
)

type MessageEvent struct {
	Name                string
	UserIDs             []string
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

func (me *MessageEvent) MessageEventHandler(event <-chan struct{}) {
	<-event
	log.Printf("EVENT: %s, TIME: %s", me.Name, me.EmittedAt.Format(time.RFC1123))
	msg, err := json.Marshal(message{
		Message:     me.Message,
		MessageType: me.Name,
	})
	if err != nil {
		log.Printf("Unable to marshal message: %v", err)
		return
	}

	go me.NotificationService.PushNotification(me.UserIDs, msg)
}
