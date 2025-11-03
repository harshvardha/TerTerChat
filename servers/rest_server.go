package servers

import (
	"context"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/harshvardha/TerTerChat/controllers"
	"github.com/harshvardha/TerTerChat/middlewares"
	"github.com/harshvardha/TerTerChat/utility"
)

func StartRESTApiServer(port string, apiConfig *controllers.ApiConfig, quit <-chan os.Signal, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Printf("[REST SERVER]: Starting server")

	// define all the handlers here
	router := http.NewServeMux()
	router.HandleFunc("GET /api/v1/healthz", func(w http.ResponseWriter, r *http.Request) {
		utility.RespondWithJson(w, http.StatusOK, "OK")
	})

	// api endpoints for authentication
	router.HandleFunc("POST /api/v1/auth/otp/send", apiConfig.HandleSendOTP)
	router.HandleFunc("POST /api/v1/auth/otp/send/registeredPhonenumber", middlewares.ValidateJWT(apiConfig.HandleSendOTPTORegisteredPhonenumber, apiConfig.JwtSecret, apiConfig.DB))
	router.HandleFunc("POST /api/v1/auth/register", apiConfig.HandleRegisterUser)
	router.HandleFunc("POST /api/v1/auth/login", apiConfig.HandleLoginUser)

	// api endpoints for users
	router.HandleFunc("PUT /api/v1/users/update/username", middlewares.ValidateJWT(apiConfig.UpdateUsername, apiConfig.JwtSecret, apiConfig.DB))
	router.HandleFunc("PUT /api/v1/users/update/phonenumber", middlewares.ValidateJWT(apiConfig.UpdatePhonenumber, apiConfig.JwtSecret, apiConfig.DB))
	router.HandleFunc("PUT /api/v1/users/update/password", middlewares.ValidateJWT(apiConfig.UpdatePassword, apiConfig.JwtSecret, apiConfig.DB))
	router.HandleFunc("GET /api/v1/users/info", middlewares.ValidateJWT(apiConfig.GetUserByPhonenumber, apiConfig.JwtSecret, apiConfig.DB))
	router.HandleFunc("DELETE /api/v1/users/remove", middlewares.ValidateJWT(apiConfig.RemoveUser, apiConfig.JwtSecret, apiConfig.DB))

	// api endpoints for messages
	router.HandleFunc("POST /api/v1/message/create", middlewares.ValidateJWT(apiConfig.HandleCreateNewMessage, apiConfig.JwtSecret, apiConfig.DB))
	router.HandleFunc("PUT /api/v1/message/update", middlewares.ValidateJWT(apiConfig.HandleUpdateMessage, apiConfig.JwtSecret, apiConfig.DB))
	router.HandleFunc("DELETE /api/v1/message/delete", middlewares.ValidateJWT(apiConfig.HandleDeleteMessage, apiConfig.JwtSecret, apiConfig.DB))
	router.HandleFunc("GET /api/v1/message/conversation", middlewares.ValidateJWT(apiConfig.HandleGetConversation, apiConfig.JwtSecret, apiConfig.DB))
	router.HandleFunc("DELETE /api/v1/message/conversation/delete", middlewares.ValidateJWT(apiConfig.HandleDeleteConversation, apiConfig.JwtSecret, apiConfig.DB))
	router.HandleFunc("GET /api/v1/message/conversations", middlewares.ValidateJWT(apiConfig.HandleGetAllConversations, apiConfig.JwtSecret, apiConfig.DB))
	router.HandleFunc("GET /api/v1/message/group/all", middlewares.ValidateJWT(apiConfig.HandleGetAllGroupMessages, apiConfig.JwtSecret, apiConfig.DB))
	router.HandleFunc("PUT /api/v1/message/mark/received", middlewares.ValidateJWT(apiConfig.HandleMarkMessageReceived, apiConfig.JwtSecret, apiConfig.DB))
	router.HandleFunc("PUT /api/v1/message/mark/read", middlewares.ValidateJWT(apiConfig.HandleMarkMessageRead, apiConfig.JwtSecret, apiConfig.DB))
	router.HandleFunc("PUT /api/v1/message/group/mark/received", middlewares.ValidateJWT(apiConfig.HandleMarkGroupMessageReceived, apiConfig.JwtSecret, apiConfig.DB))
	router.HandleFunc("PUT /api/v1/message/group/mark/read", middlewares.ValidateJWT(apiConfig.HandleMarkGroupMessageRead, apiConfig.JwtSecret, apiConfig.DB))

	// api endpoints for group
	router.HandleFunc("POST /api/v1/group/create", middlewares.ValidateJWT(apiConfig.HandleCreateGroup, apiConfig.JwtSecret, apiConfig.DB))
	router.HandleFunc("PUT /api/v1/group/update", middlewares.ValidateJWT(apiConfig.HandleUpdateGroupName, apiConfig.JwtSecret, apiConfig.DB))
	router.HandleFunc("DELETE /api/v1/group/remove", middlewares.ValidateJWT(apiConfig.HandleRemoveGroup, apiConfig.JwtSecret, apiConfig.DB))
	router.HandleFunc("GET /api/v1/group/members", middlewares.ValidateJWT(apiConfig.HandleGetAllMembersOfGroup, apiConfig.JwtSecret, apiConfig.DB))
	router.HandleFunc("PUT /api/v1/group/add/user", middlewares.ValidateJWT(apiConfig.HandleAddUserToGroup, apiConfig.JwtSecret, apiConfig.DB))
	router.HandleFunc("PUT /api/v1/group/member/remove", middlewares.ValidateJWT(apiConfig.HandleRemoveUserFromGroup, apiConfig.JwtSecret, apiConfig.DB))
	router.HandleFunc("PUT /api/v1/group/make/user/admin", middlewares.ValidateJWT(apiConfig.HandleMakeUserAdmin, apiConfig.JwtSecret, apiConfig.DB))
	router.HandleFunc("PUT /api/v1/group/remove/user/admin", middlewares.ValidateJWT(apiConfig.HandleRemoveUserFromAdmin, apiConfig.JwtSecret, apiConfig.DB))

	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
		// ReadTimeout:  time.Second * 5,
		// WriteTimeout: time.Second * 10,
		// IdleTimeout:  time.Second * 120,
	}

	// creating a server error channel to shutdown server if an unexpected error occurs during ListenAndServer()
	serverErr := make(chan error, 1)

	// launching ListenAndServe in a separate go-routine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[REST SERVER]: server failed %v", err)
			serverErr <- err
		}
	}()

	// waiting for quit signal or serverErr channel to throw ErrServerClosed
	select {
	case sig := <-quit:
		// recieved a signal(for e.g., ctrl+c or SIGTERM)
		log.Printf("[REST SERVER]: signal recieved %s. shutting down server", sig)

		// shutting down otpCache monitoring
		apiConfig.TwilioConfig.StopCacheMonitoring()

		// shutting down message cache monitoring
		apiConfig.MessageCache.StopCacheMonitoring()

		// initiating proper http server shutdown
		// creating a context with a timeout of 30 sec
		// which makes sure that shutdown process exits after 30 sec
		context, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel() // releasing all the resources acquired by the context when it done

		if err := server.Shutdown(context); err != nil {
			log.Fatalf("[REST SERVER]: server shutdown failed %v", err)
		}

		log.Printf("[REST SERVER]: server shutdown successfull")
	case err := <-serverErr:
		log.Fatalf("[REST SERVER]: server failed to start or crashed %v", err)
	}
}
