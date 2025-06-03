package controllers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/harshvardha/TerTerChat/internal/database"
	"github.com/harshvardha/TerTerChat/utility"
	"golang.org/x/crypto/bcrypt"
)

func (apiConfig *ApiConfig) HandleSendOTP(w http.ResponseWriter, r *http.Request) {
	// extracting phonenumber from request body
	type phonenumber struct {
		Phonenumber string `json:"phonenumber"`
	}
	decoder := json.NewDecoder(r.Body)
	params := phonenumber{}
	err := decoder.Decode(&params)
	if err != nil {
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// validating phonenumber
	if err = apiConfig.DataValidator.Var(params.Phonenumber, "required,phonenumber"); err != nil {
		utility.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// sending otp to phonenumber
	err = apiConfig.TwilioConfig.SendOTP(params.Phonenumber)
	if err != nil {
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utility.RespondWithJson(w, http.StatusOK, nil)
}

func (apiConfig *ApiConfig) RegisterUser(w http.ResponseWriter, r *http.Request) {
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
	if err != nil {
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// validating phonenumber
	if err = apiConfig.DataValidator.Var(params.Phonenumber, "required,phonenumber"); err != nil {
		utility.RespondWithError(w, http.StatusNotAcceptable, err.Error())
		return
	}

	// validating otp
	if err = apiConfig.TwilioConfig.VerifyOTP(params.Phonenumber, params.OTP); err != nil {
		utility.RespondWithError(w, http.StatusNotAcceptable, err.Error())
		return
	}

	// checking if the user with the phonenumber already exists
	userExists, err := apiConfig.DB.DoesUserExist(r.Context(), params.Phonenumber)
	if err != nil {
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if userExists == 1 {
		utility.RespondWithError(w, http.StatusBadRequest, "user already exist")
		return
	}

	// validating username
	if err = apiConfig.DataValidator.Var(params.Username, "required,min=4,max=50,username"); err != nil {
		utility.RespondWithError(w, http.StatusNotAcceptable, err.Error())
		return
	}

	// validating password
	if err = apiConfig.DataValidator.Var(params.Password, "required,min=8,max=20,password"); err != nil {
		utility.RespondWithError(w, http.StatusNotAcceptable, err.Error())
		return
	}

	// hashing password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(params.Password), bcrypt.DefaultCost)
	if err != nil {
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
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utility.RespondWithJson(w, http.StatusCreated, newUser)
}

func (apiConfig *ApiConfig) LoginUser(w http.ResponseWriter, r *http.Request) {
	// extracting user credentials from request body
	type userCredentials struct {
		Phonenumber string `json:"phonenumber"`
		Password    string `json:"password"`
	}

	decoder := json.NewDecoder(r.Body)
	params := userCredentials{}
	err := decoder.Decode(&params)
	if err != nil {
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// validating phonenumber
	if err = apiConfig.DataValidator.Var(params.Phonenumber, "required,phonenumber"); err != nil {
		utility.RespondWithError(w, http.StatusNotAcceptable, err.Error())
		return
	}

	// checking if the user exist with this phonenumber or not
	user, err := apiConfig.DB.GetUserByPhonenumber(r.Context(), params.Phonenumber)
	if err != nil {
		utility.RespondWithError(w, http.StatusNotFound, err.Error())
		return
	}

	// validating password
	if err = apiConfig.DataValidator.Var(params.Password, "required,min=8,max=20,password"); err != nil {
		utility.RespondWithError(w, http.StatusNotAcceptable, err.Error())
		return
	}

	// comparing password
	if err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(params.Password)); err != nil {
		utility.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// creating access token
	accessToken, err := MakeJWT(user.ID.String(), apiConfig.JwtSecret, time.Hour)
	if err != nil {
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// generating refresh token
	refreshToken, err := generateRefreshToken()
	if err != nil {
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err = apiConfig.DB.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
		Token:     refreshToken,
		UserID:    user.ID,
		ExpiresAt: time.Now().UTC().Add(time.Hour * 24 * 60),
	}); err != nil {
		utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utility.RespondWithJson(w, http.StatusOK, EmptyResponse{
		AccessToken: accessToken,
	})
}

func (apiConfig *ApiConfig) ResendOTP(w http.ResponseWriter, r *http.Request) {

}

func MakeJWT(userID string, jwtSecret string, expiresAfter time.Duration) (string, error) {
	// creating the signing key to be used for signing token
	signingKey := []byte(jwtSecret)

	// creating token claims
	tokenClaims := jwt.RegisteredClaims{
		Issuer:    "http://localhost:8080",
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresAfter).UTC()),
		Subject:   "user_id:" + userID,
	}

	// generating access token
	accessToken := jwt.NewWithClaims(jwt.SigningMethodES512, tokenClaims)

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
