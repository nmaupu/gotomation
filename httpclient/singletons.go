package httpclient

import (
	"net/url"

	"github.com/nmaupu/gotomation/logging"
	"github.com/nmaupu/gotomation/model"
	"github.com/nmaupu/gotomation/model/config"
)

var (
	// WebSocketClientSingleton is the main program acting as a daemon
	WebSocketClientSingleton *WebSocketClient
	// SimpleClientSingleton is the configuration to make simple API calls
	SimpleClientSingleton *SimpleClient
)

// Init inits all httpclient singletons
func Init(config config.Gotomation) {
	l := logging.NewLogger("Init")

	l.Debug().Msg("Creating WebSocketClientSingleton")
	WebSocketClientSingleton = NewWebSocketClient(
		model.HassConfig{
			URL:   url.URL{Scheme: "wss", Host: config.HomeAssistant.Host, Path: "api/websocket"},
			Token: config.HomeAssistant.Token,
		})

	l.Debug().Msg("Creating SimpleClientSingleton")
	SimpleClientSingleton = NewSimpleClient(
		model.HassConfig{
			URL:   url.URL{Scheme: "https", Host: config.HomeAssistant.Host, Path: "api"},
			Token: config.HomeAssistant.Token,
		})
}
