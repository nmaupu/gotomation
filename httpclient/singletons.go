package httpclient

import (
	"net/url"

	"github.com/nmaupu/gotomation/logging"
	"github.com/nmaupu/gotomation/model"
)

var (
	// WebSocketClientSingleton is the main program acting as a daemon
	WebSocketClientSingleton *WebSocketClient
	// SimpleClientSingleton is the configuration to make simple API calls
	SimpleClientSingleton *SimpleClient
)

// Init inits all httpclient singletons
func Init(hassHost, hassToken string) {
	l := logging.NewLogger("Init")

	l.Debug().Msg("Creating WebSocketClientSingleton")
	WebSocketClientSingleton = NewWebSocketClient(
		model.HassConfig{
			URL:   url.URL{Scheme: "wss", Host: hassHost, Path: "api/websocket"},
			Token: hassToken,
		})

	l.Debug().Msg("Creating SimpleClientSingleton")
	SimpleClientSingleton = NewSimpleClient(
		model.HassConfig{
			URL:   url.URL{Scheme: "https", Host: hassHost, Path: "api"},
			Token: hassToken,
		})
}
