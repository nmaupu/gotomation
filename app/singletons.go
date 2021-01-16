package app

import (
	"log"

	"github.com/nmaupu/gotomation/httpclient"
	"github.com/nmaupu/gotomation/model/config"
)

var (
	// Gotomation config
	gotomationConfig *config.Gotomation
)

// Init inits all app's singletons
func Init(conf config.Gotomation) {
	httpclient.InitSingletons(conf)
}

// GetWebSocketClient returns the daemon object
func GetWebSocketClient() *httpclient.WebSocketClient {
	return httpclient.WebSocketClientSingleton
}

// GetSimpleClient returns the hass HTTP config object to make simple API calls
func GetSimpleClient() *httpclient.SimpleClient {
	return httpclient.SimpleClientSingleton
}

// GetGotomationConfig returns the Gotomation configuration
func GetGotomationConfig() *config.Gotomation {
	return gotomationConfig
}

// Start starts the daemon
func Start() {
	if err := httpclient.WebSocketClientSingleton.Start(); err != nil {
		log.Fatal(err)
	}
}
