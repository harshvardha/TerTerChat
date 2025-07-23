package controllers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/harshvardha/TerTerChat/internal/database"
	"github.com/harshvardha/TerTerChat/utility"
	"golang.org/x/crypto/bcrypt"
)

// endpoint: /api/v1/auth/otp/send
func (apiConfig *ApiConfig) HandleSendOTP(w http.ResponseWriter, r *http.Request) {
	// extracting phonenumber from request body
	decoder := json.NewDecoder(r.Body)
	params := phonenumber{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("[/api/v1/auth/otp/send]: error decoding request body %v", err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// validating phonenumber
	if err = apiConfig.DataValidator.Var(params.Phonenumber, "required,phonenumber"); err != nil {
		log.Printf("[/api/v1/auth/otp/send]: error validating phonenumber %v", err)
		utility.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// sending otp to phonenumber
	err = apiConfig.TwilioConfig.SendOTP(params.Phonenumber)
	if err != nil {
		log.Printf("[/api/v1/auth/otp/send]: error sending otp %v", err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utility.RespondWithJson(w, http.StatusOK, nil)
}

// endpoint: /api/v1/auth/register
func (apiConfig *ApiConfig) HandleRegisterUser(w http.ResponseWriter, r *http.Request) {
	// extracting user information from request body
	type userInformation struct {
		Username    string `json:"username"`
		Phonenumber string `json:"phonenumber"`
		Password    string `json:"password"`
		OTP         string `json:"otp"`
	}

	decoder := json.NewDecoder(r.Body)
	params := userInformation{}
	err := decoder.Decode(&params)
	fmt.Println(params.Username)
	fmt.Println(params.Phonenumber)
	fmt.Println(params.Password)
	fmt.Println(params.OTP)
	if err != nil {
		log.Printf("[/api/v1/auth/register]: error decoding request body %v", err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// validating phonenumber
	if err = apiConfig.DataValidator.Var(params.Phonenumber, "required,phonenumber"); err != nil {
		log.Printf("[/api/v1/auth/register]: error validating phonenumber %v", err)
		utility.RespondWithError(w, http.StatusNotAcceptable, err.Error())
		return
	}

	// validating otp
	if err = apiConfig.TwilioConfig.VerifyOTP(params.Phonenumber, params.OTP); err != nil {
		log.Printf("[/api/v1/auth/register]: error while otp verificaiton %v", err)
		utility.RespondWithError(w, http.StatusNotAcceptable, err.Error())
		return
	}

	// checking if the user with the phonenumber already exists
	if _, err = apiConfig.DB.DoesUserExist(r.Context(), params.Phonenumber); err == nil {
		log.Printf("[/api/v1/auth/register]: user already exist while trying to register user with phonenumber %s", params.Phonenumber)
		utility.RespondWithError(w, http.StatusBadRequest, "user already exist")
		return
	}

	// validating username
	err = apiConfig.DataValidator.Var(params.Username, "required,min=4,max=50,username")
	if err != nil {
		log.Printf("[/api/v1/auth/register]: error while validating username %v", err)
		utility.RespondWithError(w, http.StatusNotAcceptable, err.Error())
		return
	}

	// validating password
	if err = apiConfig.DataValidator.Var(params.Password, "required,min=8,max=20,password"); err != nil {
		log.Printf("[/api/v1/auth/register]: error while validating password %v", err)
		utility.RespondWithError(w, http.StatusNotAcceptable, err.Error())
		return
	}

	// hashing password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(params.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("[/api/v1/auth/register]: error while creating hash of password %v", err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// registering user
	newUser, err := apiConfig.DB.CreateUser(r.Context(), database.CreateUserParams{
		Phonenumber: params.Phonenumber,
		Username:    params.Username,
		Password:    string(hashedPassword),
	})
	if err != nil {
		log.Printf("[/api/v1/auth/register]: error while creating new user %v", err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	type user struct {
		ID          uuid.UUID `json:"id"`
		Username    string    `json:"username"`
		Phonenumber string    `json:"phonenumber"`
		CreatedAt   string    `json:"created_at"`
		UpdatedAt   string    `json:"updated_at"`
	}

	utility.RespondWithJson(w, http.StatusCreated, user{
		ID:          newUser.ID,
		Username:    newUser.Username,
		Phonenumber: newUser.Phonenumber,
		CreatedAt:   newUser.CreatedAt.Format(time.RFC1123),
		UpdatedAt:   newUser.UpdatedAt.Format(time.RFC1123),
	})
}

// endpoint: /api/v1/auth/login
func (apiConfig *ApiConfig) HandleLoginUser(w http.ResponseWriter, r *http.Request) {
	// extracting user credentials from request body
	type userCredentials struct {
		Phonenumber string `json:"phonenumber"`
		Password    string `json:"password"`
	}

	decoder := json.NewDecoder(r.Body)
	params := userCredentials{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("[/api/v1/auth/login]: error while decoding request body %v", err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// validating phonenumber
	if err = apiConfig.DataValidator.Var(params.Phonenumber, "required,phonenumber"); err != nil {
		log.Printf("[/api/v1/auth/login]: error while validating phonenumber %v", err)
		utility.RespondWithError(w, http.StatusNotAcceptable, err.Error())
		return
	}

	// checking if the user exist with this phonenumber or not
	user, err := apiConfig.DB.GetUserByPhonenumber(r.Context(), params.Phonenumber)
	if err != nil {
		log.Printf("[/api/v1/auth/login]: error fetching user with phonenumber %s, %v", params.Phonenumber, err)
		utility.RespondWithError(w, http.StatusNotFound, err.Error())
		return
	}

	// validating password
	if err = apiConfig.DataValidator.Var(params.Password, "required,min=8,max=20,password"); err != nil {
		log.Printf("[/api/v1/auth/login]: error validating password %v", err)
		utility.RespondWithError(w, http.StatusNotAcceptable, err.Error())
		return
	}

	// comparing password
	if err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(params.Password)); err != nil {
		log.Printf("[/api/v1/auth/login]: error incorrect password %v", err)
		utility.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// creating access token
	accessToken, err := MakeJWT(user.ID.String(), apiConfig.JwtSecret, time.Hour)
	if err != nil {
		log.Printf("[/api/v1/auth/login]: error creating access token for user %s, %v", user.ID.String(), err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// generating refresh token
	refreshToken, err := generateRefreshToken()
	if err != nil {
		log.Printf("[/api/v1/auth/login]: error generating refresh token for user %s, %v", user.ID.String(), err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err = apiConfig.DB.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
		Token:     refreshToken,
		UserID:    user.ID,
		ExpiresAt: time.Now().UTC().Add(time.Hour * 24 * 60),
	}); err != nil {
		log.Printf("[/api/v1/auth/login]: error saving refresh token for user %s in database %v", user.ID.String(), err)
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	type latestMessages struct {
		OneToOneMessages []database.GetLatestMessagesByRecieverIDRow   `json:"one_to_messages,omitempty"`
		GroupMessages    []database.GetLatestGroupMessagesByGroupIDRow `json:"group_messages,omitempty"`
		AccessToken      string                                        `json:"access_token"`
	}

	// getting latest messages for one-to-one conversations
	newMessages := latestMessages{}
	latestOneToOneMessages, err := apiConfig.DB.GetLatestMessagesByRecieverID(r.Context(), database.GetLatestMessagesByRecieverIDParams{
		RecieverID: uuid.NullUUID{
			UUID:  user.ID,
			Valid: true,
		},
		CreatedAt: user.LastAvailable.Time,
	})
	if err == nil {
		newMessages.OneToOneMessages = latestOneToOneMessages
	} else {
		log.Printf("[/api/v1/auth/login]: no new one-to-one messages found")
	}

	// getting latest group messages
	latestGroupMessages, err := apiConfig.DB.GetLatestGroupMessagesByGroupID(r.Context(), user.LastAvailable.Time)
	if err == nil {
		newMessages.GroupMessages = latestGroupMessages
	} else {
		log.Printf("[/api/v1/auth/login]: no new group messages found")
	}

	newMessages.AccessToken = accessToken

	utility.RespondWithJson(w, http.StatusOK, newMessages)
}

func MakeJWT(userID string, jwtSecret string, expiresAfter time.Duration) (string, error) {
	// creating the signing key to be used for signing token
	signingKey := []byte(jwtSecret)

	// creating token claims
	tokenClaims := jwt.RegisteredClaims{
		Issuer:    "http://localhost:8080",
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(expiresAfter)),
		Subject:   "user_id:" + userID,
	}

	// generating access token
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS512, tokenClaims)

	// signing access token
	signedAccessToken, err := accessToken.SignedString(signingKey)
	if err != nil {
		return "", err
	}

	return signedAccessToken, nil
}

func generateRefreshToken() (string, error) {
	refreshToken := make([]byte, 32)
	_, err := rand.Read(refreshToken)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(refreshToken), nil
}
