package eventhandlers

import (
	"log"
	"net"
	"sync"
	"time"

	"github.com/harshvardha/TerTerChat/internal/services"
)

type ConnectionEvent struct {
	Name                string
	UserID              string
	ConnectionInstance  net.Conn
	NotificationService *services.Notification
	EmittedAt           time.Time
}

const (
	CONNECTED    = "CONNECTED"
	DISCONNECTED = "DISCONNECTED"
)

func ConnectionEventHandler(event chan ConnectionEvent, wg *sync.WaitGroup) {
	defer wg.Done()
	for connectionEvent := range event {
		switch connectionEvent.Name {
		case CONNECTED:
			go connectionEvent.NotificationService.AddUserConnection(connectionEvent.UserID, connectionEvent.ConnectionInstance)
		case DISCONNECTED:
			go connectionEvent.NotificationService.RemoveUserConnection(connectionEvent.UserID)
		}
	}

	log.Printf("[EVENT]: ConnectionEventHandler stopped for %s because event channel was closed, [TIME]: %s", (<-event).UserID, time.Now().Format(time.RFC1123))
}
