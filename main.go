package main

import (
	"database/sql"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/go-playground/validator/v10"
	"github.com/harshvardha/TerTerChat/controllers"
	eventhandlers "github.com/harshvardha/TerTerChat/event_handlers"
	"github.com/harshvardha/TerTerChat/internal/cache"
	"github.com/harshvardha/TerTerChat/internal/database"
	"github.com/harshvardha/TerTerChat/internal/services"
	"github.com/harshvardha/TerTerChat/servers"
	"github.com/harshvardha/TerTerChat/utility"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	// loading tcp port env variable
	tcpPort := os.Getenv("TCP_PORT")
	if tcpPort == "" {
		log.Fatal("TCP port not set")
	}

	// loading rest api port env variable
	restApiPort := os.Getenv("REST_API_PORT")
	if restApiPort == "" {
		log.Fatal("REST api port not set")
	}

	// loading jwt secret env variable
	jwtSecret := os.Getenv("ACCESS_TOKEN_SECRET")
	if jwtSecret == "" {
		log.Fatal("ACCESS_TOKEN_SECRET not set")
	}

	// loading database uri
	databaseURI := os.Getenv("DATABASE_URI")
	if databaseURI == "" {
		log.Fatal("DATABASE_URI not set")
	}

	// loading twilio account sid variable
	twilioAccountSID := os.Getenv("TWILIO_ACCOUNT_SID")
	if twilioAccountSID == "" {
		log.Fatal("TWILIO_ACCOUNT_SID not set")
	}

	// loading twilio auth token variable
	twilioAuthToken := os.Getenv("TWILIO_AUTH_TOKEN")
	if twilioAuthToken == "" {
		log.Fatal("TWILIO_AUTH_TOKEN not set")
	}

	// loading twilio verify service sid variable
	twilioVerifyServiceSID := os.Getenv("VERIFY_SERVICE_SID")
	if twilioVerifyServiceSID == "" {
		log.Fatal("VERIFY_SERVICE_SID not set")
	}

	// loading custom message variable
	customMessage := os.Getenv("CUSTOM_MESSAGE")
	if customMessage == "" {
		log.Fatal("CUSTOM_MESSAGE not set")
	}

	// loading custom friendly name variable
	customFriendlyName := os.Getenv("CUSTOM_FRIENDLY_NAME")
	if customFriendlyName == "" {
		log.Fatal("CUSTOM_FRIENDLY_NAME not set")
	}

	// setting twilio config
	twilioConfig := services.NewOTPService(
		twilioAccountSID,
		twilioVerifyServiceSID,
		twilioAuthToken,
		customMessage,
		customFriendlyName,
	)

	// creating database connection
	dbConnection, err := sql.Open("postgres", databaseURI)
	if err != nil {
		log.Fatal("Error connecting to database: ", err)
	}
	db := database.New(dbConnection)

	// registering custom validators
	dataValidator := validator.New()
	dataValidator.RegisterValidation("password", utility.PasswordValidator)
	dataValidator.RegisterValidation("username", utility.UsernameAndGroupnameValidator)
	dataValidator.RegisterValidation("groupname", utility.UsernameAndGroupnameValidator)
	dataValidator.RegisterValidation("phonenumber", utility.PhonenumberValidator)

	// communication channel for message event handler and rest api server
	messageEventEmitterChannel := make(chan eventhandlers.MessageEvent)

	// communication channel for group actions event handler and rest api server
	groupActionsEventEmitterChannel := make(chan eventhandlers.GroupEvent)

	// communication channel for connection event handler and tcp server
	connectionEventEmitterChannel := make(chan eventhandlers.ConnectionEvent)

	// notification service for pushing real time updates to users based on events
	notificationService := services.NewNotificaitonService()

	// setting up the apiConfig struct for REST server
	apiConfig := controllers.ApiConfig{
		DB:                              db,
		JwtSecret:                       jwtSecret,
		TwilioConfig:                    twilioConfig,
		NotificationService:             notificationService,
		DataValidator:                   dataValidator,
		MessageEventEmitterChannel:      messageEventEmitterChannel,
		GroupActionsEventEmitterChannel: groupActionsEventEmitterChannel,
		MessageCache:                    &cache.DynamicShardedCache[[]database.Message]{},
	}

	var wg sync.WaitGroup

	// launching message event handler
	wg.Add(1)
	go eventhandlers.MessageEventHandler(messageEventEmitterChannel, &wg)

	// launching group actions event handler
	wg.Add(1)
	go eventhandlers.GroupActionsEventHandler(groupActionsEventEmitterChannel, &wg)

	// launching connections event handler
	wg.Add(1)
	go eventhandlers.ConnectionEventHandler(connectionEventEmitterChannel, &wg)

	// creating a quit channel to listen for os signal for shutting down servers
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// starting tcp server
	wg.Add(1)
	go servers.StartTCPServer(tcpPort, notificationService, connectionEventEmitterChannel, quit, &wg)

	// starting rest api server
	wg.Add(1)
	go servers.StartRESTApiServer(restApiPort, &apiConfig, quit, &wg)

	// waiting for servers to shutdown
	<-quit
	log.Println("Shutting down servers...")
	wg.Wait()
}
