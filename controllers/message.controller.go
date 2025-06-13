package controllers

import (
	"context"
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

	// adding new message to cache
	if len(params.GroupID) > 0 {
		apiConfig.MessageCache.Set(params.GroupID, newMessage)
	} else {
		apiConfig.MessageCache.Set(userID.String()+params.ReceiverID, newMessage)
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
	senderUsername, err := apiConfig.DB.GetUserById(r.Context(), newMessage.SenderID)
	if err != nil {
		log.Printf("[/api/v1/message/create]: error fetching sender username: %v", err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	messageEvent.Message = eventhandlers.Message{
		ID:             newMessage.ID,
		Description:    newMessage.Description,
		SenderID:       newMessage.SenderID,
		SenderUsername: senderUsername.Username,
		GroupID:        newMessage.GroupID.UUID,
		CreatedAt:      newMessage.CreatedAt.Format(time.RFC1123),
	}

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

func (apiConfig *ApiConfig) UpdateMessage(w http.ResponseWriter, r *http.Request, userID uuid.UUID, newAccessToken string) {
	type request struct {
		ID          uuid.UUID `json:"id"`
		Description string    `json:"description"`
		ReceiverID  uuid.UUID `json:"receiver_id"`
		GroupID     uuid.UUID `json:"group_id"`
	}

	type response struct {
		ID          string `json:"id"`
		Description string `json:"description"`
		UpdatedAt   string `json:"updated_at"`
		AccessToken string `json:"access_token"`
	}

	// extracting request body
	decoder := json.NewDecoder(r.Body)
	params := request{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("[/api/v1/message/update]: error decoding request body: %v", err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// validating message description
	if len(params.Description) == 0 {
		log.Printf("[/api/v1/message/update]: empty message description")
		utility.RespondWithError(w, http.StatusNotAcceptable, "empty message description")
		return
	}

	// updating message
	message := database.UpdateMessageParams{
		ID:          params.ID,
		Description: params.Description,
		SenderID:    userID,
	}

	if params.GroupID != uuid.Nil {
		message.GroupID.UUID = params.GroupID
		message.GroupID.Valid = true
	}

	updatedMessage, err := apiConfig.DB.UpdateMessage(r.Context(), message)
	if err != nil {
		log.Printf("[/api/v1/message/update]: error updating message: %v", err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// updating message cache
	if params.GroupID != uuid.Nil {
		apiConfig.MessageCache.Update(params.GroupID.String(), params.ID, updatedMessage.Description, updatedMessage.Recieved, updatedMessage.UpdatedAt)
	} else {
		apiConfig.MessageCache.Update(userID.String()+params.ReceiverID.String(), params.ID, updatedMessage.Description, updatedMessage.Recieved, updatedMessage.UpdatedAt)
	}

	// creating message event
	messageEvent := eventhandlers.MessageEvent{}
	messageEvent.Name = eventhandlers.EDIT_MESSAGE

	// fetching contact numbers of group members
	if params.GroupID != uuid.Nil {
		receiversPhonenumbers, err := apiConfig.DB.GetGroupMembersPhonenumbers(r.Context(), params.GroupID)
		if err != nil {
			log.Printf("[/api/v1/message/update]: error fetching phonenumbers of group members: %v", err)
			utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		messageEvent.Phonenumbers = receiversPhonenumbers
	}

	// fetching contact number of receiver
	if params.ReceiverID != uuid.Nil {
		receiverPhonenumber, err := apiConfig.DB.GetUserPhonenumberByID(r.Context(), params.ReceiverID)
		if err != nil {
			log.Printf("[/api/v1/message/update]: error fetching receiver phonenumber: %v", err)
			utility.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}

		messageEvent.Phonenumbers = []string{receiverPhonenumber}
	}

	// adding message to messageEvent
	sender, err := apiConfig.DB.GetUserById(r.Context(), updatedMessage.SenderID)
	if err != nil {
		log.Printf("[/api/v1/message/edit]: error fetching sender: %v", err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	messageEvent.Message = eventhandlers.Message{
		ID:             params.ID,
		Description:    updatedMessage.Description,
		SenderID:       updatedMessage.SenderID,
		SenderUsername: sender.Username,
		GroupID:        updatedMessage.GroupID.UUID,
		UpdatedAt:      updatedMessage.UpdatedAt.Format(time.RFC1123),
	}

	// adding notification service
	messageEvent.NotificationService = apiConfig.NotificationService

	// adding event emitting time
	messageEvent.EmittedAt = time.Now()

	// emitting the event
	apiConfig.MessageEventEmitterChannel <- messageEvent

	utility.RespondWithJson(w, http.StatusOK, response{
		ID:          params.ID.String(),
		Description: updatedMessage.Description,
		UpdatedAt:   updatedMessage.UpdatedAt.Format(time.RFC1123),
		AccessToken: newAccessToken,
	})
}

func (apiConfig *ApiConfig) DeleteMessage(w http.ResponseWriter, r *http.Request, userID uuid.UUID, newAccessToken string) {
	type request struct {
		ID         uuid.UUID `json:"id"`
		ReceiverID uuid.UUID `json:"receiver_id"`
		GroupID    uuid.UUID `json:"group_id"`
	}

	// extracting request body
	decoder := json.NewDecoder(r.Body)
	params := request{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("[/api/v1/message/delete]: error decoding request body: %v", err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// validating request body
	if params.ID == uuid.Nil {
		log.Printf("[/api/v1/message/delete]: empty message id")
		utility.RespondWithError(w, http.StatusNotAcceptable, "empty message id")
		return
	}

	if params.ReceiverID == uuid.Nil && params.GroupID == uuid.Nil {
		log.Printf("[/api/v1/message/delete]: empty receiver id and group id")
		utility.RespondWithError(w, http.StatusNotAcceptable, "empty receiver id and group id")
		return
	}

	// deleting message
	deleteMessageParams := database.RemoveMessageParams{
		ID:       params.ID,
		SenderID: userID,
	}
	if params.ReceiverID != uuid.Nil {
		deleteMessageParams.RecieverID.UUID = params.ReceiverID
		deleteMessageParams.RecieverID.Valid = true
	}
	if params.GroupID != uuid.Nil {
		deleteMessageParams.GroupID.UUID = params.GroupID
		deleteMessageParams.GroupID.Valid = true
	}

	deletedMessage, err := apiConfig.DB.RemoveMessage(r.Context(), deleteMessageParams)
	if err != nil {
		log.Printf("[/api/v1/message/delete]: error deleting message: %v", err)
		utility.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// removing message from cache
	if params.GroupID != uuid.Nil {
		apiConfig.MessageCache.RemoveMessage(params.GroupID.String(), params.ID)
	} else {
		apiConfig.MessageCache.RemoveMessage(userID.String()+params.ReceiverID.String(), params.ID)
	}

	// creating delete_message event
	messageEvent := eventhandlers.MessageEvent{}
	messageEvent.Name = eventhandlers.DELETE_MESSAGE

	// fetching contact number of group members
	if params.GroupID != uuid.Nil {
		phonenumbers, err := fetchGroupMembersContacts(params.GroupID, r.Context(), apiConfig.DB)
		if err != nil {
			log.Printf("[/api/v1/message/delete]: error fetching group members contacts: %v", err)
			utility.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}

		messageEvent.Phonenumbers = phonenumbers
	}

	// fetching contact number of receiver
	if params.ReceiverID != uuid.Nil {
		phonenumber, err := fetchReceiverContactNumber(params.ReceiverID, r.Context(), apiConfig.DB)
		if err != nil {
			log.Printf("[/api/v1/message/delete]: error fetching receiver contact: %v", err)
			utility.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}

		messageEvent.Phonenumbers = []string{phonenumber}
	}

	// adding message to messageEvent
	messageEvent.Message = eventhandlers.Message{
		ID:       params.ID,
		SenderID: deletedMessage.SenderID,
		GroupID:  deletedMessage.GroupID.UUID,
	}

	// adding notification service
	messageEvent.NotificationService = apiConfig.NotificationService

	// adding event emitting time
	messageEvent.EmittedAt = time.Now()

	// emitting event
	apiConfig.MessageEventEmitterChannel <- messageEvent

	utility.RespondWithJson(w, http.StatusOK, EmptyResponse{
		AccessToken: newAccessToken,
	})
}

func (apiConfig *ApiConfig) GetConversation(w http.ResponseWriter, r *http.Request, userID uuid.UUID, newAccessToken string) {
	type request struct {
		ReceiverID uuid.UUID `json:"receiver_id"`
		CreatedAt  time.Time `json:"created_at"`
	}

	type response struct {
		Messages    []database.Message `json:"messages"`
		AccessToken string             `json:"access_token"`
	}

	// extracting request body
	decoder := json.NewDecoder(r.Body)
	params := request{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("[/api/v1/message/conversation]: error decoding request body: %v", err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// validating request body
	if params.ReceiverID == uuid.Nil {
		log.Printf("[/api/v1/message/conversation]: empty receiver id")
		utility.RespondWithError(w, http.StatusBadRequest, "empty receiver id")
		return
	}

	if params.CreatedAt.IsZero() {
		log.Printf("[/api/v1/message/conversation]: empty created_at time")
		utility.RespondWithError(w, http.StatusBadRequest, "empty created at time")
		return
	}

	// first checking if the messages are present in cache
	// if messages are not present in cache then hitting database
	// fetching group messages with limit 10 sorted in ascending order by created_at
	var messages []database.Message
	messages = apiConfig.MessageCache.Get(userID.String()+params.ReceiverID.String(), params.CreatedAt)
	if messages != nil {
		utility.RespondWithJson(w, http.StatusOK, response{
			Messages:    messages,
			AccessToken: newAccessToken,
		})
	}

	messages, err = apiConfig.DB.GetAllMessages(r.Context(), database.GetAllMessagesParams{
		SenderID: userID,
		RecieverID: uuid.NullUUID{
			UUID:  params.ReceiverID,
			Valid: true,
		},
		CreatedAt: params.CreatedAt,
	})
	if err != nil {
		log.Printf("[/api/v1/message/conversation]: error fetching messages: %v", err)
		utility.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	utility.RespondWithJson(w, http.StatusOK, response{
		Messages:    messages,
		AccessToken: newAccessToken,
	})
}

func (apiConfig *ApiConfig) GetAllGroupMessages(w http.ResponseWriter, r *http.Request, userID uuid.UUID, newAccessToken string) {
	type request struct {
		GroupID   uuid.UUID `json:"group_id"`
		CreatedAt time.Time `json:"created_at"`
	}

	type response struct {
		Messages    []database.Message `json:"messages"`
		AccessToken string             `json:"access_token"`
	}

	// extracting request body
	decoder := json.NewDecoder(r.Body)
	params := request{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("[/api/v1/message/group]: error decoding request body: %v", err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// validating request body
	if params.GroupID == uuid.Nil {
		log.Printf("[/api/v1/message/group]: empty group id")
		utility.RespondWithError(w, http.StatusNotAcceptable, "empty group id")
		return
	}

	if params.CreatedAt.IsZero() {
		log.Printf("[/api/v1/message/group]: empty created at time")
		utility.RespondWithError(w, http.StatusNotAcceptable, "empty created at time")
		return
	}

	// first checking if the messages are present in cache
	// if messages are not present in cache then hitting database
	// fetching group messages with limit 10 sorted in ascending order by created_at
	var messages []database.Message
	messages = apiConfig.MessageCache.Get(params.GroupID.String(), params.CreatedAt)
	if messages != nil {
		utility.RespondWithJson(w, http.StatusOK, response{
			Messages:    messages,
			AccessToken: newAccessToken,
		})
	}

	messages, err = apiConfig.DB.GetAllGroupMessages(r.Context(), database.GetAllGroupMessagesParams{
		GroupID: uuid.NullUUID{
			UUID:  params.GroupID,
			Valid: true,
		},
		CreatedAt: params.CreatedAt,
	})
	if err != nil {
		log.Printf("[/api/v1/message/group]: error fetching messages for group %s: %v", params.GroupID, err)
		utility.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	utility.RespondWithJson(w, http.StatusOK, response{
		Messages:    messages,
		AccessToken: newAccessToken,
	})
}

func (apiConfig *ApiConfig) MarkMessageReceived(w http.ResponseWriter, r *http.Request, userID uuid.UUID, newAccessToken string) {
	type request struct {
		MessageID uuid.UUID `json:"message_id"`
		SenderID  uuid.UUID `json:"sender_id"`
	}

	// extracting request body
	decoder := json.NewDecoder(r.Body)
	params := request{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("[/api/v1/message/received]: error decoding request body: %v", err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// marking the message as received
	updatedAt, err := apiConfig.DB.MarkMessageReceived(r.Context(), database.MarkMessageReceivedParams{
		ID: params.MessageID,
		RecieverID: uuid.NullUUID{
			UUID:  userID,
			Valid: true,
		},
		SenderID: params.SenderID,
	})
	if err != nil {
		log.Printf("[/api/v1/message/received]: error marking message received: %v", err)
		utility.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// updating cache
	apiConfig.MessageCache.Update(params.SenderID.String()+userID.String(), params.MessageID, "", true, updatedAt)

	// creating message event
	messageEvent := eventhandlers.MessageEvent{}
	messageEvent.Name = eventhandlers.MESSAGE_RECEIVED

	// fetching receiver phonenumber
	senderContact, err := fetchReceiverContactNumber(userID, r.Context(), apiConfig.DB)
	if err != nil {
		log.Printf("[/api/v1/message/received]: error fetching sender phonenumber: %v", err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	messageEvent.Phonenumbers = []string{senderContact}

	// adding message to messageEvent
	messageEvent.Message = eventhandlers.Message{
		ID:         params.MessageID,
		ReceiverID: userID,
	}

	// adding notification service
	messageEvent.NotificationService = apiConfig.NotificationService

	// adding event emitting time
	messageEvent.EmittedAt = time.Now()

	// emitting event
	apiConfig.MessageEventEmitterChannel <- messageEvent

	utility.RespondWithJson(w, http.StatusOK, EmptyResponse{
		AccessToken: newAccessToken,
	})
}

func (apiConfig *ApiConfig) MarkGroupMessageRead(w http.ResponseWriter, r *http.Request, userID uuid.UUID, newAccessToken string) {
	type request struct {
		MessageID     uuid.UUID `json:"message_id"`
		GroupID       uuid.UUID `json:"group_id"`
		GroupMemberID uuid.UUID `json:"group_member_id"`
	}

	// extracting request body
	decoder := json.NewDecoder(r.Body)
	params := request{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("[/api/v1/message/group/read]: error decoding request body: %v", err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// marking group message read
	if err = apiConfig.DB.MarkGroupMessageRead(r.Context(), database.MarkGroupMessageReadParams{
		MessageID:     params.MessageID,
		GroupMemberID: params.GroupMemberID,
		GroupID:       params.GroupID,
	}); err != nil {
		log.Printf("[/api/v1/message/group/read]: error marking group message read: %v", err)
		utility.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// creating message event
	messageEvent := eventhandlers.MessageEvent{}
	messageEvent.Name = eventhandlers.GROUP_MESSAGE_READ

	// fetching groupmembers phonenumbers
	groupMembersPhonenumbers, err := fetchGroupMembersContacts(params.GroupID, r.Context(), apiConfig.DB)
	if err != nil {
		log.Printf("[/api/v1/message/group/read]: error fetching group members phonenumbers: %v", err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	messageEvent.Phonenumbers = groupMembersPhonenumbers

	// adding Message to message event
	messageReader, err := apiConfig.DB.GetUserById(r.Context(), params.GroupMemberID) // messageReader is the member who just read the message
	if err != nil {
		log.Printf("[/api/v1/message/group/read]: error fetching message reader: %v", err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	messageEvent.Message = eventhandlers.Message{
		ID:              params.MessageID,
		GroupID:         params.GroupID,
		GroupMemberID:   params.GroupMemberID,
		GroupMemberName: messageReader.Username,
	}

	// adding notification service to message event
	messageEvent.NotificationService = apiConfig.NotificationService

	// adding event emitting time
	messageEvent.EmittedAt = time.Now()

	// emitting event
	apiConfig.MessageEventEmitterChannel <- messageEvent

	utility.RespondWithJson(w, http.StatusOK, EmptyResponse{
		AccessToken: newAccessToken,
	})
}

func fetchGroupMembersContacts(groupID uuid.UUID, ctx context.Context, db *database.Queries) ([]string, error) {
	contactNumbers, err := db.GetGroupMembersPhonenumbers(ctx, groupID)
	return contactNumbers, err
}

func fetchReceiverContactNumber(receiverID uuid.UUID, ctx context.Context, db *database.Queries) (string, error) {
	contactNumber, err := db.GetUserPhonenumberByID(ctx, receiverID)
	return contactNumber, err
}
