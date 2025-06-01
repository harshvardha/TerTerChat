package services

import (
	"net"
	"sync"
)

type Notification struct {
	connections map[string]net.Conn
	mutex       sync.RWMutex
}

func (conn *Notification) PushNotification(user_ids []string, message []byte) {
	conn.mutex.RLock()
	defer conn.mutex.RUnlock()

	for _, id := range user_ids {
		connection := conn.connections[id]
		connection.Write(message)
	}
}

func (conn *Notification) AddUserConnection(userID string, connection net.Conn) {
	conn.mutex.Lock()
	defer conn.mutex.Unlock()

	conn.connections[userID] = connection
}

func (conn *Notification) RemoveUserConnection(userID string) {
	conn.mutex.Lock()
	defer conn.mutex.Unlock()

	delete(conn.connections, userID)
}
