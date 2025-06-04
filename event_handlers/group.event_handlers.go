package eventhandlers

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/harshvardha/TerTerChat/internal/services"
)

type GroupEvent struct {
	Name                string
	GroupID             string
	UserIDs             []string
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
	Name      string   `json:"name"`
	GroupID   string   `json:"groupID"`
	UserIDs   []string `json:"userIDs"`
	EmittedAt string   `json:"emittedAt"`
}

func GroupActionsEventHandler(event chan GroupEvent, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Println("[EVENT_HANDLER]: group actions event handler started")
	for groupEvent := range event {
		log.Printf("[EVENT]: %s, [TIME]: %s", groupEvent.Name, groupEvent.EmittedAt.Format(time.RFC1123))
		action, err := json.Marshal(action{
			Name:      groupEvent.Name,
			GroupID:   groupEvent.GroupID,
			UserIDs:   groupEvent.UserIDs,
			EmittedAt: groupEvent.EmittedAt.Format(time.RFC1123),
		})
		if err != nil {
			log.Printf("[EVENT]: Unable to marshal group event action %v", err)
			return
		}

		go groupEvent.NotificationService.PushNotification(groupEvent.UserIDs, action)
	}

	log.Printf("[EVENT]: GroupActionsEventHandler stopped for %v because event channel was closed, [TIME]: %s", (<-event).UserIDs, time.Now().Format(time.RFC1123))
}
