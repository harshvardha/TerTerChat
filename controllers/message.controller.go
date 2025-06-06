package controllers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	eventhandlers "github.com/harshvardha/TerTerChat/event_handlers"
	"github.com/harshvardha/TerTerChat/internal/database"
	"github.com/harshvardha/TerTerChat/utility"
)

func (apiConfig *ApiConfig) CreateNewMessage(w http.ResponseWriter, r *http.Request, userID uuid.UUID, newAccessToken string) {
	type request struct {
		Description string `json:"description"`
		ReceiverID  string `json:"receiver_id"`
		GroupID     string `json:"group_id"`
	}

	type response struct {
		ID          string `json:"id"`
		Description string `json:"description"`
		UpdatedAt   string `json:"updated_at"`
		AccessToken string `json:"accessToken"`
	}

	// extracting message from request body
	decoder := json.NewDecoder(r.Body)
	params := request{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("[/api/v1/message/create]: error decoding request body: %v", err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// validating message body
	if (params.ReceiverID == "" && params.GroupID == "") || (len(params.ReceiverID) > 0 && len(params.GroupID) > 0) {
		log.Printf("[/api/v1/message/create]: invalid message body")
		utility.RespondWithError(w, http.StatusNotAcceptable, "invalid message body")
		return
	}

	if params.Description == "" {
		log.Printf("[/api/v1/message/create]: empty message description")
		utility.RespondWithError(w, http.StatusNotAcceptable, "empty message description")
		return
	}

	// creating new message
	message := database.CreateMessageParams{}
	message.Description = params.Description

	if len(params.ReceiverID) > 0 {
		receiverId, err := uuid.Parse(params.ReceiverID)
		if err != nil {
			log.Printf("[/api/v1/message/create]: error parsing the message receiver id: %v", err)
			utility.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		message.RecieverID = uuid.NullUUID{
			UUID:  receiverId,
			Valid: true,
		}
	}

	if len(params.GroupID) > 0 {
		groupId, err := uuid.Parse(params.GroupID)
		if err != nil {
			log.Printf("[/api/v1/message/create]: error parsing the message group id: %v", err)
			utility.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		message.GroupID = uuid.NullUUID{
			UUID:  groupId,
			Valid: true,
		}
	}

	message.Sent = true

	newMessage, err := apiConfig.DB.CreateMessage(r.Context(), message)
	if err != nil {
		log.Printf("[/api/v1/message/create]: error creating new message: %v", err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// emitting new message event
	messageEvent := eventhandlers.MessageEvent{}

	// adding event name
	messageEvent.Name = eventhandlers.NEW_MESSAGE

	// adding the receivers contact number
	if newMessage.GroupID.UUID != uuid.Nil {
		groupMembersID, err := apiConfig.DB.GetGroupMembersPhonenumbers(r.Context(), newMessage.GroupID.UUID)
		if err != nil {
			log.Printf("[/api/v1/message/create]: error fetching group members id: %v", err)
			utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		messageEvent.Phonenumbers = groupMembersID
	}

	if newMessage.RecieverID.UUID != uuid.Nil {
		receiverPhonenumber, err := apiConfig.DB.GetUserPhonenumberByID(r.Context(), newMessage.RecieverID.UUID)
		if err != nil {
			log.Printf("[/api/v1/message/create]: error fetching the receiver phonenumber: %v", err)
			utility.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}

		messageEvent.Phonenumbers = []string{receiverPhonenumber}
	}

	// adding the message
	messageEvent.Message = newMessage

	// providing event handler the instance of notification service
	messageEvent.NotificationService = apiConfig.NotificationService

	// providing event emitting time instance
	messageEvent.EmittedAt = time.Now()

	// passing the event to event handler
	apiConfig.MessageEventEmitterChannel <- messageEvent

	utility.RespondWithJson(w, http.StatusCreated, response{
		ID:          newMessage.ID.String(),
		Description: newMessage.Description,
		UpdatedAt:   newMessage.UpdatedAt.Format(time.RFC1123),
		AccessToken: newAccessToken,
	})
}

func (apiConfig *ApiConfig) UpdateMessage(w http.ResponseWriter, r *http.Request, userID string, newAccessToken string) {

}

func (apiConfig *ApiConfig) DeleteMessage(w http.ResponseWriter, r *http.Request, userID string, newAccessToken string) {

}

func (apiConfig *ApiConfig) GetConversation(w http.ResponseWriter, r *http.Request, userID string, newAccessToken string) {

}

func (apiConfig *ApiConfig) GetAllGroupMessages(w http.ResponseWriter, r *http.Request, userID string, newAccessToken string) {

}
