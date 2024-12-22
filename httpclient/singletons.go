package httpclient

import (
	"net/url"
	"sync"

	"github.com/nmaupu/gotomation/logging"
	"github.com/nmaupu/gotomation/model"
)

var (
	mutexSimpleClient sync.Mutex
	scs               SimpleClient

	mutexWebSocketClient sync.Mutex
	wsc                  WebSocketClient
)

// InitSimpleClient inits the SimpleClient singleton
func InitSimpleClient(scheme string, hassHost, hassToken string, healthCheckEntities []model.HassEntity) {
	l := logging.NewLogger("initSimpleClient")
	mutexSimpleClient.Lock()
	defer mutexSimpleClient.Unlock()

	l.Debug().Msg("Creating SimpleClient singleton")
	scs = NewSimpleClient(
		model.HassConfig{
			URL:                 url.URL{Scheme: scheme, Host: hassHost, Path: "api"},
			Token:               hassToken,
			HealthCheckEntities: healthCheckEntities,
		})
}

// GetSimpleClient returns the SimpleClient singleton
func GetSimpleClient() SimpleClient {
	mutexSimpleClient.Lock()
	defer mutexSimpleClient.Unlock()
	return scs
}

// InitWebSocketClient inits the WebSocketClient singleton
func InitWebSocketClient(scheme, hassHost, hassToken string) {
	l := logging.NewLogger("initWebSocketClient")

	mutexWebSocketClient.Lock()
	defer mutexWebSocketClient.Unlock()

	if wsc != nil {
		wsc.Stop()
		wsc = nil
	}

	l.Debug().Msg("Creating WebSocketClient singleton")
	wsc = NewWebSocketClient(
		model.HassConfig{
			URL:   url.URL{Scheme: scheme, Host: hassHost, Path: "api/websocket"},
			Token: hassToken,
		})
}

// GetWebSocketClient returns the WebSocketClient singleton
func GetWebSocketClient() WebSocketClient {
	mutexWebSocketClient.Lock()
	defer mutexWebSocketClient.Unlock()
	return wsc
}

func IsConnectedAndAuthenticated() bool {
	mutexWebSocketClient.Lock()
	defer mutexWebSocketClient.Unlock()
	return wsc.Authenticated() && wsc.Connected()
}
