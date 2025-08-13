package eventhandlers

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/harshvardha/TerTerChat/internal/services"
)

type Message struct {
	ID              uuid.UUID
	Description     string
	SenderID        uuid.UUID
	ReceiverID      uuid.UUID
	GroupID         uuid.UUID
	SenderUsername  string
	GroupMemberID   uuid.UUID
	GroupMemberName string
	CreatedAt       string
	UpdatedAt       string
}

// Message data for NEW_MESSAGE | EDIT_MESSAGE event
type newOrEditMessage struct {
	ID             uuid.UUID `json:"id"`
	GroupID        uuid.UUID `json:"group_id,omitempty"`
	SenderID       uuid.UUID `json:"sender_id"`
	SenderUsername string    `json:"sender_username,omitempty"`
	Description    string    `json:"description"`
	CreatedAt      string    `json:"created_at,omitempty"`
	UpdatedAt      string    `json:"updated_at,omitempty"`
}

// Message data for DELETE_MESSAGE event
type deleteMessage struct {
	ID       uuid.UUID `json:"id"`
	SenderID uuid.UUID `json:"sender_id"`
	GroupID  uuid.UUID `json:"group_id,omitempty"`
}

// Message data for MESSAGE_RECEIVED event
type markMessageReceived struct {
	ID         uuid.UUID `json:"id"`
	ReceiverID uuid.UUID `json:"receiver_id"`
}

// Message data for MESSAGE_READ event
type markMessageRead struct {
	ID         uuid.UUID `json:"id"`
	SenderID   uuid.UUID `json:"sender_id"`
	ReceiverID uuid.UUID `json:"receiver_id"`
}

// Message data for GROUP_MESSAGE_READ and GROUP_MESSAGE_RECEIVED event
type markGroupMessageReadOrReceived struct {
	ID                  uuid.UUID `json:"id"`
	GroupID             uuid.UUID `json:"group_id"`
	GroupMemberID       uuid.UUID `json:"group_member_id"`
	GroupMemberUsername string    `json:"group_member_username"`
}

const (
	NEW_MESSAGE            = "NEW_MESSAGE"
	EDIT_MESSAGE           = "EDIT_MESSAGE"
	DELETE_MESSAGE         = "DELETE_MESSAGE"
	MESSAGE_RECEIVED       = "MARK_MESSAGE_RECEIVED"
	MESSAGE_READ           = "MARK_MESSAGE_READ"
	GROUP_MESSAGE_RECEIVED = "GROUP_MESSAGE_RECEIVED"
	GROUP_MESSAGE_READ     = "GROUP_MESSAGE_READ"
)

type MessageEvent struct {
	Name                string
	Phonenumbers        []string
	Message             Message
	NotificationService *services.Notification
	EmittedAt           time.Time
}

/*
This event handler will first check which event has been emitted then accordingly will
create the response byte by using the respective instance of the response structs defined above
Structure of the response sent by this event handler:

	it will be a byte array containing information about the event that will be relevant to client
	the byte array is divided into three parts:
		1. event name
		2. separator pipe '|'
		3. instance of response struct
*/
func MessageEventHandler(event chan MessageEvent, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Printf("[MESSAGE_EVENT_HANDLER]: message event handler started")
	for messageEvent := range event {
		log.Printf("[MESSAGE_EVENT_HANDLER]: %s", messageEvent.Name)

		// event name will be added to final []byte type response
		// and they will be separated by a '|'(pipe) character
		// so that interpreting on client side becomes easier
		eventNameByte := []byte(messageEvent.Name)
		separator := []byte("|")
		offset := 0 // offset will be used for sub-indexing the response byte slice when we create the final response

		switch messageEvent.Name {
		case NEW_MESSAGE:
			msg, err := json.Marshal(newOrEditMessage{
				ID:             messageEvent.Message.ID,
				GroupID:        messageEvent.Message.GroupID,
				SenderID:       messageEvent.Message.SenderID,
				SenderUsername: messageEvent.Message.SenderUsername,
				Description:    messageEvent.Message.Description,
				CreatedAt:      messageEvent.Message.CreatedAt,
			})
			if err != nil {
				log.Printf("[MESSAGE_EVENT_HANDLER]: error marshalling json for new_message: %v", err)
				continue
			}

			// final response
			response := make([]byte, len(eventNameByte)+len(msg)+1)

			// copying the event name into response
			copy(response[offset:], eventNameByte)
			offset += len(eventNameByte)

			// copying the byte for separator
			copy(response[offset:], separator)
			offset++

			// copying the message
			copy(response[offset:], msg)

			messageEvent.NotificationService.PushNotification(messageEvent.Phonenumbers, response)
		case EDIT_MESSAGE:
			msg, err := json.Marshal(newOrEditMessage{
				ID:          messageEvent.Message.ID,
				GroupID:     messageEvent.Message.GroupID,
				SenderID:    messageEvent.Message.SenderID,
				Description: messageEvent.Message.Description,
				UpdatedAt:   messageEvent.Message.UpdatedAt,
			})
			if err != nil {
				log.Printf("[MESSAGE_EVENT_HANDLER]: error marshalling json for edit_message: %v", err)
				continue
			}

			// final response
			response := make([]byte, len(eventNameByte)+len(msg)+1)

			// copying the event name into response
			copy(response[offset:], eventNameByte)
			offset += len(eventNameByte)

			// copying the byte for separator
			copy(response[offset:], separator)
			offset++

			// copying the message
			copy(response[offset:], msg)

			messageEvent.NotificationService.PushNotification(messageEvent.Phonenumbers, response)
		case DELETE_MESSAGE:
			msg, err := json.Marshal(deleteMessage{
				ID:       messageEvent.Message.ID,
				SenderID: messageEvent.Message.SenderID,
				GroupID:  messageEvent.Message.GroupID,
			})
			if err != nil {
				log.Printf("[MESSAGE_EVENT_HANDLER]: error marshalling json for delete_message: %v", err)
				continue
			}

			// final response
			response := make([]byte, len(eventNameByte)+len(msg)+1)

			// copying the event name into response
			copy(response[offset:], eventNameByte)
			offset += len(eventNameByte)

			// copying the byte for separator
			copy(response[offset:], separator)
			offset++

			// copying the message
			copy(response[offset:], msg)

			messageEvent.NotificationService.PushNotification(messageEvent.Phonenumbers, response)
		case MESSAGE_RECEIVED:
			msg, err := json.Marshal(markMessageReceived{
				ID:         messageEvent.Message.ID,
				ReceiverID: messageEvent.Message.ReceiverID,
			})
			if err != nil {
				log.Printf("[MESSAGE_EVENT_HANDLER]: error marshalling json for message_received: %v", err)
				continue
			}

			// final response
			response := make([]byte, len(eventNameByte)+len(msg)+1)

			// copying the event name into response
			copy(response[offset:], eventNameByte)
			offset += len(eventNameByte)

			// copying the byte for separator
			copy(response[offset:], separator)
			offset++

			// copying the message
			copy(response[offset:], msg)

			messageEvent.NotificationService.PushNotification(messageEvent.Phonenumbers, response)
		case MESSAGE_READ:
			msg, err := json.Marshal(markMessageRead{
				ID:         messageEvent.Message.ID,
				SenderID:   messageEvent.Message.SenderID,
				ReceiverID: messageEvent.Message.ReceiverID,
			})
			if err != nil {
				log.Printf("[MESSAGE_EVENT_HANDLER]: error marshalling json for MESSAGE_READ event: %v", err)
				continue
			}

			// final response
			response := make([]byte, len(eventNameByte)+len(msg)+1)

			// copying the event name into response
			copy(response[offset:], eventNameByte)
			offset += len(eventNameByte)

			// copying the byte for separator
			copy(response[offset:], separator)
			offset++

			// copying the message
			copy(response[offset:], msg)

			messageEvent.NotificationService.PushNotification(messageEvent.Phonenumbers, response)
		case GROUP_MESSAGE_RECEIVED:
			msg, err := json.Marshal(markGroupMessageReadOrReceived{
				ID:                  messageEvent.Message.ID,
				GroupID:             messageEvent.Message.GroupID,
				GroupMemberID:       messageEvent.Message.GroupMemberID,
				GroupMemberUsername: messageEvent.Message.GroupMemberName,
			})
			if err != nil {
				log.Printf("[MESSAGE_EVENT_HANDLER]: error marshalling json for GROUP_MESSAGE_RECEIVED event: %v", err)
				continue
			}

			// final response
			response := make([]byte, len(msg)+len(eventNameByte)+1)

			// copying the event name into response
			copy(response[offset:], eventNameByte)

			// copying the byte for separator
			copy(response[offset:], separator)

			// copying the message byte
			copy(response[offset:], msg)

			messageEvent.NotificationService.PushNotification(messageEvent.Phonenumbers, response)
		case GROUP_MESSAGE_READ:
			msg, err := json.Marshal(markGroupMessageReadOrReceived{
				ID:                  messageEvent.Message.ID,
				GroupID:             messageEvent.Message.GroupID,
				GroupMemberID:       messageEvent.Message.GroupMemberID,
				GroupMemberUsername: messageEvent.Message.GroupMemberName,
			})
			if err != nil {
				log.Printf("[MESSAGE_EVENT_HANDLER]: error marshalling json for group_message_read: %v", err)
				continue
			}

			// final response
			response := make([]byte, len(eventNameByte)+len(msg)+1)

			// copying the event name into response
			copy(response[offset:], eventNameByte)
			offset += len(eventNameByte)

			// copying the byte for separator
			copy(response[offset:], separator)
			offset++

			// copying the message
			copy(response[offset:], msg)

			messageEvent.NotificationService.PushNotification(messageEvent.Phonenumbers, response)
		}

	}

	log.Printf("[MESSAGE_EVENT_HANDLER]: Message event handler for %v stopped because event channel was closed", (<-event).Phonenumbers)
}
