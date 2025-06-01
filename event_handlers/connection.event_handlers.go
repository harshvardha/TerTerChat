package eventhandlers

import (
	"net"
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

func (ce *ConnectionEvent) ConnectionEventHandler(event <-chan struct{}) {
	<-event
	switch ce.Name {
	case CONNECTED:
		go ce.NotificationService.AddUserConnection(ce.UserID, ce.ConnectionInstance)
	case DISCONNECTED:
		go ce.NotificationService.RemoveUserConnection(ce.UserID)
	}
}
