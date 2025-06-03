package middlewares

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/harshvardha/TerTerChat/controllers"
	"github.com/harshvardha/TerTerChat/internal/database"
	"github.com/harshvardha/TerTerChat/utility"
)

func getSubjects(tokenSubject string) (map[string]string, error) {
	subjects := strings.Split(tokenSubject, ",")
	if len(subjects) == 0 {
		return nil, errors.New("no subject found")
	}

	subjectMap := make(map[string]string, len(subjects))
	for _, subject := range subjects {
		pair := strings.Split(subject, ":")
		if len(pair) == 0 {
			return nil, errors.New("not all claims found")
		}
		subjectMap[pair[0]] = pair[1]
	}

	return subjectMap, nil
}

type authenticatedEndpointHandler func(http.ResponseWriter, *http.Request, uuid.UUID, string)

func ValidateJWT(handler authenticatedEndpointHandler, tokenSecret string, db *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := strings.Split(r.Header.Get("Authorization"), " ")
		if len(authHeader) != 2 {
			utility.RespondWithError(w, http.StatusUnauthorized, "malformed request header")
			return
		}

		jwtClaims := jwt.RegisteredClaims{}
		token, parseError := jwt.ParseWithClaims(authHeader[1], jwtClaims, func(token *jwt.Token) (any, error) {
			return []byte(tokenSecret), nil
		})

		subjects, err := token.Claims.GetSubject()
		if err != nil {
			utility.RespondWithError(w, http.StatusUnauthorized, err.Error())
			return
		}

		parsedSubjects, err := getSubjects(subjects)
		if err != nil {
			utility.RespondWithError(w, http.StatusUnauthorized, err.Error())
			return
		}

		userID, err := uuid.Parse(parsedSubjects["user_id"])
		if err != nil {
			utility.RespondWithError(w, http.StatusNotAcceptable, err.Error())
			return
		}

		if parseError != nil {
			if errors.Is(parseError, jwt.ErrTokenExpired) {
				refreshTokenExpiresAt, err := db.GetRefreshTokenExpirationTime(r.Context(), userID)
				if err != nil {
					utility.RespondWithError(w, http.StatusUnauthorized, err.Error())
					return
				}

				if time.Now().After(refreshTokenExpiresAt) {
					utility.RespondWithError(w, http.StatusUnauthorized, "Please login again")
					return
				} else {
					newAccessToken, err := controllers.MakeJWT(userID.String(), tokenSecret, time.Hour)
					if err != nil {
						utility.RespondWithError(w, http.StatusInternalServerError, err.Error())
						return
					}

					handler(w, r, userID, newAccessToken)
				}
			}

			utility.RespondWithError(w, http.StatusUnauthorized, parseError.Error())
			return
		}

		handler(w, r, userID, "")
	}
}
