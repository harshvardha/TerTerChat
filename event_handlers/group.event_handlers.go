package eventhandlers

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/harshvardha/TerTerChat/internal/services"
)

type Group struct {
	ID          uuid.UUID
	Username    string
	Phonenumber string
}

type GroupEvent struct {
	Name                string
	Group               Group
	Phonenumbers        []string
	NotificationService *services.Notification
	EmittedAt           time.Time
}

const (
	ADD_USER_TO_GROUP      = "ADD_USER_TO_GROUP"
	REMOVE_USER_FROM_GROUP = "REMOVE_USER_FROM_GROUP"
	MADE_ADMIN             = "MADE_ADMIN"
	REMOVE_ADMIN           = "REMOVE_ADMIN"
)

type action struct {
	Name      string `json:"name"`
	Group     Group  `json:"groupID"`
	EmittedAt string `json:"emittedAt"`
}

func GroupActionsEventHandler(event chan GroupEvent, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Println("[GROUP_EVENT_HANDLER]: event handler started")
	for groupEvent := range event {
		log.Printf("[GROUP_EVENT_HANDLER]: %s event", groupEvent.Name)
		eventNameByte := []byte(groupEvent.Name)
		separatorByte := []byte("|")
		offset := 0
		action, err := json.Marshal(action{
			Name:      groupEvent.Name,
			Group:     groupEvent.Group,
			EmittedAt: groupEvent.EmittedAt.Format(time.RFC1123),
		})
		if err != nil {
			log.Printf("[GROUP_EVENT_HANDLER]: Unable to marshal group event action %v", err)
			return
		}

		response := make([]byte, len(eventNameByte)+len(action)+1)

		copy(response[offset:], eventNameByte)
		offset += len(eventNameByte)

		copy(response[offset:], separatorByte)
		offset++

		copy(response[offset:], action)

		go groupEvent.NotificationService.PushNotification(groupEvent.Phonenumbers, action)
	}

	log.Printf("[GROUP_EVENT_HANDLER]: stopped for %v because event channel was closed", (<-event).Phonenumbers)
}
