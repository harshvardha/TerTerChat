package eventhandlers

import (
	"encoding/json"
	"log"
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

func (ge *GroupEvent) GroupActionsEventHandler(event <-chan struct{}) {
	<-event
	log.Printf("EVENT: %s, TIME: %s", ge.Name, ge.EmittedAt.Format(time.RFC1123))
	action, err := json.Marshal(action{
		Name:      ge.Name,
		GroupID:   ge.GroupID,
		UserIDs:   ge.UserIDs,
		EmittedAt: ge.EmittedAt.Format(time.RFC1123),
	})
	if err != nil {
		log.Printf("Unable to marshal group event action %v", err)
		return
	}

	go ge.NotificationService.PushNotification(ge.UserIDs, action)
}
