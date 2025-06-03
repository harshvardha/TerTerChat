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
	Phonenumber         string
	ConnectionInstance  net.Conn
	NotificationService *services.Notification
	EmittedAt           time.Time
}

const (
	CONNECTED      = "CONNECTED"
	DISCONNECTED   = "DISCONNECTED"
	LAST_AVAILABLE = "LAST_AVAILABLE"
)

// TODO: add a last_available event handler
func ConnectionEventHandler(event chan ConnectionEvent, wg *sync.WaitGroup) {
	defer wg.Done()
	for connectionEvent := range event {
		switch connectionEvent.Name {
		case CONNECTED:
			go connectionEvent.NotificationService.AddUserConnection(connectionEvent.Phonenumber, connectionEvent.ConnectionInstance)
		case DISCONNECTED:
			go connectionEvent.NotificationService.RemoveUserConnection(connectionEvent.Phonenumber)
		}
	}

	log.Printf("[EVENT]: ConnectionEventHandler stopped for %s because event channel was closed, [TIME]: %s", (<-event).Phonenumber, time.Now().Format(time.RFC1123))
}
