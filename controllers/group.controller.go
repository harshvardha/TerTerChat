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

// endpoint: /api/v1/groups/create
func (apiConfig *ApiConfig) HandleCreateGroup(w http.ResponseWriter, r *http.Request, userID uuid.UUID, newAccessToken string) {
	type request struct {
		Name string `json:"name"`
	}

	type response struct {
		ID          uuid.UUID `json:"id"`
		Name        string    `json:"name"`
		CreatedAt   string    `json:"created_at"`
		AccessToken string    `json:"access_token"`
	}

	// extracting request body
	decoder := json.NewDecoder(r.Body)
	params := request{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("[/api/v1/group/create]: error decoding request body: %v", err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// validating request body
	if len(params.Name) == 0 {
		log.Printf("[/api/v1/group/create]: empty name field")
		utility.RespondWithError(w, http.StatusNotAcceptable, "empty name field")
		return
	}

	// creating group and making the requesting user admin
	newGroup, err := apiConfig.DB.CreateGroup(r.Context(), params.Name)
	if err != nil {
		log.Printf("[/api/v1/group/create]: error creating new group: %v", err)
		utility.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err = apiConfig.DB.MakeUserAdmin(r.Context(), database.MakeUserAdminParams{
		UserID:  userID,
		GroupID: newGroup.ID,
	}); err != nil {
		log.Printf("[/api/v1/group/create]: error making the requesting user admin: %v", err)
		if removeGroupErr := apiConfig.DB.DeleteGroup(r.Context(), newGroup.ID); removeGroupErr != nil {
			log.Printf("[/api/v1/group/create]: error removing group: %v", err)
		}

		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utility.RespondWithJson(w, http.StatusCreated, response{
		ID:          newGroup.ID,
		Name:        newGroup.Name,
		CreatedAt:   newGroup.CreatedAt.Format(time.RFC1123),
		AccessToken: newAccessToken,
	})
}

// endpoint: /api/v1/group/update
func (apiConfig *ApiConfig) HandleUpdateGroupName(w http.ResponseWriter, r *http.Request, userID uuid.UUID, newAccessToken string) {
	type request struct {
		GroupID uuid.UUID `json:"group_id"`
		Name    string    `json:"name"`
	}

	type response struct {
		Name        string `json:"name"`
		UpdatedAt   string `json:"updated_at"`
		AccessToken string `json:"access_token"`
	}

	// extracting request body
	decoder := json.NewDecoder(r.Body)
	params := request{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("[/api/v1/group/update]: error decoding request body: %v", err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// validating request body
	if len(params.Name) == 0 {
		log.Printf("[/api/v1/group/update]: empty name field")
		utility.RespondWithError(w, http.StatusNotAcceptable, "empty name field")
		return
	}

	if params.GroupID == uuid.Nil {
		log.Printf("[/api/v1/group/update]: empty group id")
		utility.RespondWithError(w, http.StatusNotAcceptable, "empty group id")
		return
	}

	// checking if the requesting user id admin of group or not
	if _, err := apiConfig.DB.IsUserAdmin(r.Context(), database.IsUserAdminParams{
		UserID:  userID,
		GroupID: params.GroupID,
	}); err != nil {
		log.Printf("[/api/v1/group/update]: requesting user is not group admin: %v", err)
		utility.RespondWithError(w, http.StatusUnauthorized, err.Error())
		return
	}

	// updating group name
	updatedGroup, err := apiConfig.DB.UpdateGroup(r.Context(), database.UpdateGroupParams{
		Name: params.Name,
		ID:   params.GroupID,
	})
	if err != nil {
		log.Printf("[/api/v1/group/update]: error updating group name: %v", err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utility.RespondWithJson(w, http.StatusOK, response{
		Name:        updatedGroup.Name,
		UpdatedAt:   updatedGroup.UpdatedAt.Format(time.RFC1123),
		AccessToken: newAccessToken,
	})
}

/*
endpoint: /api/v1/group/remove
This endpoint helps in deleting the group
*/
func (apiConfig *ApiConfig) HandleRemoveGroup(w http.ResponseWriter, r *http.Request, userID uuid.UUID, newAccessToken string) {
	type request struct {
		GroupID uuid.UUID `json:"group_id"`
	}

	// extracting request body
	decoder := json.NewDecoder(r.Body)
	params := request{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("[/api/v1/group/remove]: error decoding request body: %v", err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// validating request body
	if params.GroupID == uuid.Nil {
		log.Printf("[/api/v1/group/remove]: empty group id field")
		utility.RespondWithError(w, http.StatusNotAcceptable, "empty group id field")
		return
	}

	// checking if the requesting user id admin or not
	if _, err := apiConfig.DB.IsUserAdmin(r.Context(), database.IsUserAdminParams{
		UserID:  userID,
		GroupID: params.GroupID,
	}); err != nil {
		log.Printf("[/api/v1/group/remove]: requesting user %s is not admin: %v", userID, err)
		utility.RespondWithError(w, http.StatusUnauthorized, err.Error())
		return
	}

	// removing group
	if err = apiConfig.DB.DeleteGroup(r.Context(), params.GroupID); err != nil {
		log.Printf("[/api/v1/group/remove]: error removing group: %v", err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utility.RespondWithJson(w, http.StatusOK, EmptyResponse{
		AccessToken: newAccessToken,
	})
}

/*
endpoint: /api/v1/group/members
this endpoint will provide list of all the members of the group
whose id is provided with request
*/
func (apiConfig *ApiConfig) HandleGetAllMembersOfGroup(w http.ResponseWriter, r *http.Request, userID uuid.UUID, newAccessToken string) {
	type request struct {
		GroupID uuid.UUID `json:"group_id"`
	}

	type response struct {
		Members     []database.GetGroupMembersRow `json:"members"`
		AccessToken string                        `json:"access_token"`
	}

	// decoding request body
	decoder := json.NewDecoder(r.Body)
	params := request{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("[/api/v1/group/members]: error decoding request body: %v", err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// validating request body
	if params.GroupID == uuid.Nil {
		log.Printf("[/api/v1/group/members]: empty group id field")
		utility.RespondWithError(w, http.StatusNotAcceptable, "empty group id field")
		return
	}

	// fetching all the group members
	groupMembers, err := apiConfig.DB.GetGroupMembers(r.Context(), database.GetGroupMembersParams{
		GroupID: params.GroupID,
		ID:      userID,
	})
	if err != nil {
		log.Printf("[/api/v1/group/members]: error fetching group members: %v", err)
		utility.RespondWithError(w, http.StatusNoContent, "group id invalid")
		return
	}

	// creating response
	utility.RespondWithJson(w, http.StatusOK, response{
		Members:     groupMembers,
		AccessToken: newAccessToken,
	})
}

/*
endpoint: /api/v1/group/add/user
This endpoint helps in adding a new member to the group
*/
func (apiConfig *ApiConfig) HandleAddUserToGroup(w http.ResponseWriter, r *http.Request, userID uuid.UUID, newAccessToken string) {
	type request struct {
		MemberPhonenumber string    `json:"member_phonenumber"`
		GroupID           uuid.UUID `json:"group_id"`
	}

	// extracting request body
	decoder := json.NewDecoder(r.Body)
	params := request{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("[/api/v1/group/user/add]: error decoding request body: %v", err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// validating request body
	if err = apiConfig.DataValidator.Var(params.MemberPhonenumber, "required,phonenumber"); err != nil {
		log.Printf("[/api/v1/group/user/add]: invalud member phonenumber: %v", err)
		utility.RespondWithError(w, http.StatusNotAcceptable, err.Error())
		return
	}

	if params.GroupID == uuid.Nil {
		log.Printf("[/api/v1/group/user/remove]: empty group id field")
		utility.RespondWithError(w, http.StatusNotAcceptable, "empty group id field")
		return
	}

	// fetching the user_id for the provided member phonenumber
	user, err := apiConfig.DB.GetUserByPhonenumber(r.Context(), params.MemberPhonenumber)
	if err != nil {
		log.Printf("[/api/v1/group/user/add]: no user found with phonenumber %s: %v", params.MemberPhonenumber, err)
		utility.RespondWithError(w, http.StatusNotFound, err.Error())
		return
	}

	// checking if the requesting user id admin or not
	if _, err := apiConfig.DB.IsUserAdmin(r.Context(), database.IsUserAdminParams{
		UserID:  userID,
		GroupID: params.GroupID,
	}); err != nil {
		log.Printf("[/api/v1/group/user/add]: requesting user is not admin: %v", err)
		utility.RespondWithError(w, http.StatusUnauthorized, err.Error())
		return
	}

	// adding user to group
	if err = apiConfig.DB.AddUserToGroup(r.Context(), database.AddUserToGroupParams{
		UserID:  user.ID,
		GroupID: params.GroupID,
	}); err != nil {
		log.Printf("[/api/v1/group/user/add]: error adding user to group: %v", err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// if user was previously part of the group then mark is_receiver_allowed_to_see = true
	apiConfig.DB.MarkIsAllowedToSeeAsTrueForSpecificGroupMember(r.Context(), database.MarkIsAllowedToSeeAsTrueForSpecificGroupMemberParams{
		MemberID: user.ID,
		GroupID:  params.GroupID,
	})

	// emit group event ADD_USER_TO_GROUP
	groupEvent := eventhandlers.GroupEvent{}
	groupEvent.Name = eventhandlers.ADD_USER_TO_GROUP

	// creating group actions
	groupEvent.Group = eventhandlers.Group{
		ID:          params.GroupID,
		Username:    user.Username,
		Phonenumber: params.MemberPhonenumber,
	}

	// fetching group members phonenumbers
	phonenumbers, err := apiConfig.DB.GetGroupMembersPhonenumbers(r.Context(), params.GroupID)
	if err != nil {
		log.Printf("[/api/v1/group/user/remove]: error fetching phonenumbers of group members: %v", err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	groupEvent.Phonenumbers = phonenumbers

	// adding notification service
	groupEvent.NotificationService = apiConfig.NotificationService

	// adding event emitting time
	groupEvent.EmittedAt = time.Now()

	// emitting event
	apiConfig.GroupActionsEventEmitterChannel <- groupEvent

	utility.RespondWithJson(w, http.StatusOK, EmptyResponse{
		AccessToken: newAccessToken,
	})
}

/*
endpoint: /api/v1/group/member/remove
This endpoint helps in removing a member from the group
*/
func (apiConfig *ApiConfig) HandleRemoveUserFromGroup(w http.ResponseWriter, r *http.Request, userID uuid.UUID, newAccessToken string) {
	type request struct {
		UserID  uuid.UUID `json:"user_id"`
		GroupID uuid.UUID `json:"group_id"`
	}

	// extracting request body
	decoder := json.NewDecoder(r.Body)
	params := request{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("[/api/v1/group/user/remove]: error decoding request body: %v", err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// validating request body
	if params.UserID == uuid.Nil {
		log.Printf("[/api/v1/group/user/remove]: empty user id field")
		utility.RespondWithError(w, http.StatusNotAcceptable, "empty user id field")
		return
	}

	if params.GroupID == uuid.Nil {
		log.Printf("[/api/v1/group/user/remove]: empty group id field")
		utility.RespondWithError(w, http.StatusNotAcceptable, "empty group id field")
		return
	}

	// checking if the requesting user id admin or not
	if _, err := apiConfig.DB.IsUserAdmin(r.Context(), database.IsUserAdminParams{
		UserID:  userID,
		GroupID: params.GroupID,
	}); err != nil {
		log.Printf("[/api/v1/group/user/remove]: requesting user is not admin: %v", err)
		utility.RespondWithError(w, http.StatusUnauthorized, err.Error())
		return
	}

	// removing user from group
	if err = apiConfig.DB.RemoveUserFromGroup(r.Context(), database.RemoveUserFromGroupParams{
		UserID:  params.UserID,
		GroupID: params.GroupID,
	}); err != nil {
		log.Printf("[/api/v1/group/user/remove]: error removing user from group: %v", err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// emit group event REMOVE_USER_FROM_GROUP
	groupEvent := eventhandlers.GroupEvent{}
	groupEvent.Name = eventhandlers.REMOVE_USER_FROM_GROUP

	// creating group action
	user, err := apiConfig.DB.GetUserById(r.Context(), params.UserID)
	if err != nil {
		log.Printf("[/api/v1/group/user/remove]: error fetching requested user information: %v", err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	groupEvent.Group = eventhandlers.Group{
		ID:          params.GroupID,
		Username:    user.Username,
		Phonenumber: user.Phonenumber,
	}

	// fetching group members phonenumbers
	phonenumbers, err := apiConfig.DB.GetGroupMembersPhonenumbers(r.Context(), params.GroupID)
	if err != nil {
		log.Printf("[/api/v1/group/user/remove]: error fetching phonenumbers of group members: %v", err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	groupEvent.Phonenumbers = phonenumbers

	// adding notification service
	groupEvent.NotificationService = apiConfig.NotificationService

	// adding event emitting time
	groupEvent.EmittedAt = time.Now()

	// emitting event
	apiConfig.GroupActionsEventEmitterChannel <- groupEvent

	utility.RespondWithJson(w, http.StatusOK, EmptyResponse{
		AccessToken: newAccessToken,
	})
}

// endpoint: /api/v1/group/make/user/admin
func (apiConfig *ApiConfig) HandleMakeUserAdmin(w http.ResponseWriter, r *http.Request, userID uuid.UUID, newAccessToken string) {
	type request struct {
		GroupID uuid.UUID `json:"group_id"`
		UserID  uuid.UUID `json:"user_id"`
	}

	// extracting request body
	decoder := json.NewDecoder(r.Body)
	params := request{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("[/api/v1/group/admin/make]: error decoding request body: %v", err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// validating request body
	if params.GroupID == uuid.Nil {
		log.Printf("[/api/v1/group/admin/make]: empty group id field")
		utility.RespondWithError(w, http.StatusNotAcceptable, "empty group id field")
		return
	}

	if params.UserID == uuid.Nil {
		log.Printf("[/api/v1/group/admin/make]: empty user id field")
		utility.RespondWithError(w, http.StatusNotAcceptable, "empty user id field")
		return
	}

	// checking if the requesting user is admin or not
	if _, err := apiConfig.DB.IsUserAdmin(r.Context(), database.IsUserAdminParams{
		UserID:  userID,
		GroupID: params.GroupID,
	}); err != nil {
		log.Printf("[/api/v1/group/admin/make]: requesting user is not admin: %v", err)
		utility.RespondWithError(w, http.StatusUnauthorized, err.Error())
		return
	}

	// making user admin
	if err = apiConfig.DB.MakeUserAdmin(r.Context(), database.MakeUserAdminParams{
		GroupID: params.GroupID,
		UserID:  params.UserID,
	}); err != nil {
		log.Printf("[/api/v1/group/admin/make]: error making user admin: %v", err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// emit group event MADE_ADMIN
	groupEvent := eventhandlers.GroupEvent{}
	groupEvent.Name = eventhandlers.MADE_ADMIN

	// creating group action
	user, err := apiConfig.DB.GetUserById(r.Context(), params.UserID)
	if err != nil {
		log.Printf("[/api/v1/group/admin/make]: error fetching requested user information: %v", err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	groupEvent.Group = eventhandlers.Group{
		ID:          params.GroupID,
		Username:    user.Username,
		Phonenumber: user.Phonenumber,
	}

	// fetching phonenumbers of group members
	phonenumbers, err := apiConfig.DB.GetGroupMembersPhonenumbers(r.Context(), params.GroupID)
	if err != nil {
		log.Printf("[/api/v1/group/admin/make]: error fetching phonenumbers for group members: %v", err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	groupEvent.Phonenumbers = phonenumbers

	// adding notification service
	groupEvent.NotificationService = apiConfig.NotificationService

	// adding event emitting time
	groupEvent.EmittedAt = time.Now()

	// emitting event
	apiConfig.GroupActionsEventEmitterChannel <- groupEvent

	utility.RespondWithJson(w, http.StatusOK, EmptyResponse{
		AccessToken: newAccessToken,
	})
}

// endpoint: /api/v1/group/remove/user/admin
func (apiConfig *ApiConfig) HandleRemoveUserFromAdmin(w http.ResponseWriter, r *http.Request, userID uuid.UUID, newAccessToken string) {
	type request struct {
		UserID  uuid.UUID `json:"user_id"`
		GroupID uuid.UUID `json:"group_id"`
	}

	// extracting request body
	decoder := json.NewDecoder(r.Body)
	params := request{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("[/api/v1/group/admin/remove]: error decoding request body: %v", err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// validating request body
	if params.UserID == uuid.Nil {
		log.Printf("[/api/v1/group/admin/remove]: empty user id field")
		utility.RespondWithError(w, http.StatusNotAcceptable, "empty user id field")
		return
	}

	if params.GroupID == uuid.Nil {
		log.Printf("[/api/v1/group/admin/remove]: empty group id field")
		utility.RespondWithError(w, http.StatusNotAcceptable, "empty group id field")
		return
	}

	// checking if the requesting user is admin or not
	if _, err := apiConfig.DB.IsUserAdmin(r.Context(), database.IsUserAdminParams{
		UserID:  userID,
		GroupID: params.GroupID,
	}); err != nil {
		log.Printf("[/api/v1/group/admin/remove]: requesting user is not admin: %v", err)
		utility.RespondWithError(w, http.StatusUnauthorized, err.Error())
		return
	}

	// removing requested user from admin
	if err = apiConfig.DB.RemoveUserFromAdmin(r.Context(), database.RemoveUserFromAdminParams{
		UserID:  params.UserID,
		GroupID: params.GroupID,
	}); err != nil {
		log.Printf("[/api/v1/group/admin/remove]: error removing requested user from admin: %v", err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// emit group event REMOVE_ADMIN
	groupEvent := eventhandlers.GroupEvent{}
	groupEvent.Name = eventhandlers.REMOVE_ADMIN

	// fetching phonenumbers of groupmembers
	phonenumbers, err := apiConfig.DB.GetGroupMembersPhonenumbers(r.Context(), params.GroupID)
	if err != nil {
		log.Printf("[/api/v1/group/admin/remove]: error fetching phonenumbers of group members: %v", err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	groupEvent.Phonenumbers = phonenumbers

	// creating group action
	user, err := apiConfig.DB.GetUserById(r.Context(), params.UserID)
	if err != nil {
		log.Printf("[/api/v1/group/admin/remove]: error fetching requested user information: %v", err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	group := eventhandlers.Group{
		ID:          params.GroupID,
		Phonenumber: user.Phonenumber,
		Username:    user.Username,
	}
	groupEvent.Group = group

	// adding notification service
	groupEvent.NotificationService = apiConfig.NotificationService

	// adding event emitting time
	groupEvent.EmittedAt = time.Now()

	// emitting event
	apiConfig.GroupActionsEventEmitterChannel <- groupEvent

	utility.RespondWithJson(w, http.StatusOK, EmptyResponse{
		AccessToken: newAccessToken,
	})
}
