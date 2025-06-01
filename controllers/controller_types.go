package controllers

import (
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
	MessageEventEmitterChannel      chan eventhandlers.MessageEvent
	GroupActionsEventEmitterChannel chan eventhandlers.GroupEvent
	MessageCache                    *cache.DynamicShardedCache[[]database.Message]
}
