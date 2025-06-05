package services

import (
	"context"
	"log"
	"net"
	"sync"
	"time"

	"github.com/harshvardha/TerTerChat/internal/database"
)

type Notification struct {
	connections map[string]net.Conn
	mutex       sync.RWMutex
}

func NewNotificaitonService() *Notification {
	log.Printf("[NOTIFICATION_SERVICE]: started notification service")
	return &Notification{
		connections: make(map[string]net.Conn),
	}
}

func (conn *Notification) PushNotification(phonenumbers []string, message []byte) {
	conn.mutex.RLock()
	defer conn.mutex.RUnlock()

	for _, phonenumber := range phonenumbers {
		connection := conn.connections[phonenumber]
		connection.Write(message)
	}
}

func (conn *Notification) AddUserConnection(phonenumber string, connection net.Conn) {
	conn.mutex.Lock()
	defer conn.mutex.Unlock()

	conn.connections[phonenumber] = connection
}

func (conn *Notification) RemoveUserConnection(phonenumber string, db *database.Queries) {
	conn.mutex.Lock()
	defer conn.mutex.Unlock()

	// marking user's last logout time
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	if err := db.SetLastAvailable(ctx, phonenumber); err != nil {
		log.Printf("[EVENT]: unable to set last available time for disconnected user: %v", err)
	}
	delete(conn.connections, phonenumber)
}
