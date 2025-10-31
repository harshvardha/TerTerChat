package controllers

import (
	"github.com/go-playground/validator/v10"
	eventhandlers "github.com/harshvardha/TerTerChat/event_handlers"
	"github.com/harshvardha/TerTerChat/internal/cache"
	"github.com/harshvardha/TerTerChat/internal/database"
	"github.com/harshvardha/TerTerChat/internal/services"
)

type ApiConfig struct {
	DB                              *database.Queries
	JwtSecret                       string
	TwilioConfig                    *services.TwilioConfig
	NotificationService             *services.Notification
	DataValidator                   *validator.Validate
	MessageEventEmitterChannel      chan eventhandlers.MessageEvent
	GroupActionsEventEmitterChannel chan eventhandlers.GroupEvent
	MessageCache                    *cache.DynamicShardedCache
}

type EmptyResponse struct {
	AccessToken string `json:"access_token"`
}

type phonenumber struct {
	Phonenumber string `json:"phonenumber"`
}
