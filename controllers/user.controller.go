package controllers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/harshvardha/TerTerChat/internal/database"
	"github.com/harshvardha/TerTerChat/utility"
	"golang.org/x/crypto/bcrypt"
)

// endpoint: /api/v1/users/update/username
func (apiConfig *ApiConfig) UpdateUsername(w http.ResponseWriter, r *http.Request, userID uuid.UUID, newAccessToken string) {
	// extracting new username from request body
	type request struct {
		Username string `json:"username"`
	}

	type response struct {
		Username    string `json:"username"`
		AccessToken string `json:"access_token"`
	}

	decoder := json.NewDecoder(r.Body)
	params := request{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("[/api/v1/users/update/username]: error decoding request body: %v", err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// validating username
	if err = apiConfig.DataValidator.Var(params.Username, "required,min=4,max=50,username"); err != nil {
		log.Printf("[/api/v1/users/update/username]: username validation failed: %v", err)
		utility.RespondWithError(w, http.StatusNotAcceptable, err.Error())
		return
	}

	// updating username
	newUsername, err := apiConfig.DB.UpdateUsername(r.Context(), database.UpdateUsernameParams{
		Username: params.Username,
		ID:       userID,
	})
	if err != nil {
		log.Printf("[/api/v1/users/update/username]: error updating username: %v", err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utility.RespondWithJson(w, http.StatusOK, response{
		Username:    newUsername,
		AccessToken: newAccessToken,
	})
}

// endpoint: /api/v1/users/update/phonenumber
func (apiConfig *ApiConfig) UpdatePhonenumber(w http.ResponseWriter, r *http.Request, userID uuid.UUID, newAccessToken string) {
	// extracting new phonenumber and otp from request body
	type request struct {
		Phonenumber string `json:"phonenumber"`
		OTP         string `json:"otp"`
	}

	decoder := json.NewDecoder(r.Body)
	params := request{}
	err := decoder.Decode(&params)
	if err != nil {
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// validating phonenumber
	if err = apiConfig.DataValidator.Var(params.Phonenumber, "required,phonenumber"); err != nil {
		log.Printf("[/api/v1/users/update/phonenumber]: error validating phonenumber: %v", err)
		utility.RespondWithError(w, http.StatusNotAcceptable, err.Error())
		return
	}

	// validating otp
	if err = apiConfig.TwilioConfig.VerifyOTP(params.Phonenumber, params.OTP); err != nil {
		log.Printf("[/api/v1/users/update/phonenumber]: error validating otp: %v", err)
		utility.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// updating phonenumber
	if err = apiConfig.DB.UpdatePhonenumber(r.Context(), database.UpdatePhonenumberParams{
		Phonenumber: params.Phonenumber,
		ID:          userID,
	}); err != nil {
		log.Printf("[/api/v1/users/update/phonenumber]: error updating phonenumber: %v", err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// deleting refresh token
	if err = apiConfig.DB.RemoveRefreshToken(r.Context(), userID); err != nil {
		log.Printf("[/api/v1/users/update/phonenumber]: error deleting refresh token: %v", err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utility.RespondWithJson(w, http.StatusOK, EmptyResponse{
		AccessToken: newAccessToken,
	})
}

// endpoint: /api/v1/users/update/password
func (apiConfig *ApiConfig) UpdatePassword(w http.ResponseWriter, r *http.Request, userID uuid.UUID, newAccessToken string) {
	// extracting new password and otp from request body
	type request struct {
		Password    string `json:"password"`
		Phonenumber string `json:"phonenumber"`
		OTP         string `json:"otp"`
	}

	decoder := json.NewDecoder(r.Body)
	params := request{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("[/api/v1/users/update/password]: error decoding request body: %v", err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// validating otp
	if err = apiConfig.TwilioConfig.VerifyOTP(params.Phonenumber, params.OTP); err != nil {
		log.Printf("[/api/v1/users/update/password]: error validating otp while updating password: %v", err)
		utility.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// validating password
	if err = apiConfig.DataValidator.Var(params.Password, "required,min=8,max=20,password"); err != nil {
		log.Printf("[/api/v1/users/update/password]: error validating password while updating it: %v", err)
		utility.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// hashing new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(params.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("[/api/v1/users/update/password]: error hashing password while updating it: %v", err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// updating password
	if err = apiConfig.DB.UpdatePassword(r.Context(), database.UpdatePasswordParams{
		Password: string(hashedPassword),
		ID:       userID,
	}); err != nil {
		log.Printf("[/api/v1/users/update/password]: error updating password: %v", err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// removing existing refresh token
	if err = apiConfig.DB.RemoveRefreshToken(r.Context(), userID); err != nil {
		log.Printf("[/api/v1/users/update/password]: error deleting refresh token: %v", err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utility.RespondWithJson(w, http.StatusOK, nil)
}

// endpoint: /api/v1/users/info
func (apiConfig *ApiConfig) GetUserByPhonenumber(w http.ResponseWriter, r *http.Request, userID uuid.UUID, newAccessToken string) {
	// extracting phonenumber from request body
	type request struct {
		Phonenumber string `json:"phonenumber"`
	}

	type response struct {
		Username    string `json:"username"`
		CreatedAt   string `json:"created_at"`
		AccessToken string `json:"access_token"`
	}

	decoder := json.NewDecoder(r.Body)
	params := request{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("[/api/v1/user/get]: error decoding request body: %v", err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// getting user info
	userInfo, err := apiConfig.DB.GetUserByPhonenumber(r.Context(), params.Phonenumber)
	if err != nil {
		log.Printf("[/api/v1/user/get]: user with phonenumber %s not found: %v", params.Phonenumber, err)
		utility.RespondWithError(w, http.StatusNotFound, err.Error())
		return
	}

	utility.RespondWithJson(w, http.StatusOK, response{
		Username:    userInfo.Username,
		CreatedAt:   userInfo.CreatedAt.Format(time.RFC1123),
		AccessToken: newAccessToken,
	})
}

// endpoint: /api/v1/users/remove
func (apiConfig *ApiConfig) RemoveUser(w http.ResponseWriter, r *http.Request, userID uuid.UUID, newAccessToken string) {
	if err := apiConfig.DB.RemoveUser(r.Context(), userID); err != nil {
		log.Printf("[/api/v1/users/remove]: error removing user account %v: %v", userID, err)
		utility.RespondWithError(w, http.StatusNotFound, err.Error())
		return
	}
}
