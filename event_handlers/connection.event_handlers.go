package eventhandlers

import (
	"log"
	"net"
	"sync"
	"time"

	"github.com/harshvardha/TerTerChat/internal/database"
	"github.com/harshvardha/TerTerChat/internal/services"
)

type ConnectionEvent struct {
	Name                string
	Phonenumber         string
	ConnectionInstance  net.Conn
	NotificationService *services.Notification
	DB                  *database.Queries
	EmittedAt           time.Time
}

const (
	CONNECTED    = "CONNECTED"
	DISCONNECTED = "DISCONNECTED"
)

// TODO for later update: add a last_available event handler so that diconnection event does not get emitted from socket server
// instead it should be emitted from rest server where first we register the last logout time of user then emit disconnection event
func ConnectionEventHandler(event chan ConnectionEvent, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Println("[EVENT_HANDLER]: connection event handler started")
	for connectionEvent := range event {
		switch connectionEvent.Name {
		case CONNECTED:
			go connectionEvent.NotificationService.AddUserConnection(connectionEvent.Phonenumber, connectionEvent.ConnectionInstance)
		case DISCONNECTED:
			go connectionEvent.NotificationService.RemoveUserConnection(connectionEvent.Phonenumber, connectionEvent.DB)
		}
	}

	log.Printf("[EVENT]: ConnectionEventHandler stopped for %s because event channel was closed, [TIME]: %s", (<-event).Phonenumber, time.Now().Format(time.RFC1123))
}
