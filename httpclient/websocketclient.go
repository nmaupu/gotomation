package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/nmaupu/gotomation/app"
	"github.com/nmaupu/gotomation/logging"
	"github.com/nmaupu/gotomation/model"
)

const (
	// ErrorCodeIDReuse is returned when id an already used ID
	ErrorCodeIDReuse = "id_reuse"
	// ErrorInvalidFormat is returned when the message is invalid or req id is 0
	ErrorInvalidFormat = "invalid_format"
)

// ResponseHandlerSignature is the callback func signature when receiving a response from the server
type ResponseHandlerSignature func(model.HassAPIObject)

// callback store the callback func and its associated concrete type
type callback struct {
	F            ResponseHandlerSignature
	ConcreteType model.HassAPIObject
}

// WebSocketClient manages a WebSocket connection to hass
type WebSocketClient struct {
	model.HassConfig
	mutexConn        sync.Mutex
	conn             net.Conn
	callbacks        map[string]callback
	EventsSubscribed map[uint64]model.HassEventSubscription
	// id to use for the next request
	id uint64

	// requestChannel is used to share WebSocketRequest objects between go routines
	requestChannel chan *WebSocketRequest
	// workerRequestsHandlerStop is used to stop workerRequestsHandler routine
	workerRequestsHandlerStop chan bool
	// workerDaemonStop is used to stop workerDaemon routine
	workerDaemonStop chan bool

	// requestsTracker keeps track of requests sent to the server waiting for a result
	requestsTracker WebSocketRequestsTracker
	authenticated   bool
}

// NewWebSocketClient returns a new NewWebSocketClient initialized
func NewWebSocketClient(config model.HassConfig) *WebSocketClient {
	return &WebSocketClient{
		HassConfig:     config,
		requestChannel: make(chan *WebSocketRequest, 10),
		// important: setting size of chan bools to 1 to avoid being blocked until read when sending the stop message
		workerRequestsHandlerStop: make(chan bool, 1),
		workerDaemonStop:          make(chan bool, 1),
	}
}

// RegisterCallback registers a new callback given its type
func (c *WebSocketClient) RegisterCallback(hassType string, f ResponseHandlerSignature, concreteType model.HassAPIObject) {
	if c.callbacks == nil {
		c.callbacks = make(map[string]callback, 0)
	}

	c.callbacks[hassType] = callback{
		F:            f,
		ConcreteType: concreteType,
	}
}

// DeregisterCallback deregisters a callback given its type
func (c *WebSocketClient) DeregisterCallback(hassType string) {
	if c.callbacks != nil {
		delete(c.callbacks, hassType)
	}
}

func (c *WebSocketClient) mustConnect(retryEvery time.Duration) {
	l := logging.NewLogger("WebSocketClient.mustConnect")
	var err error

	// forcing connection to close
	c.closeConn()
	c.authenticated = false
	atomic.StoreUint64(&c.id, 0)

	for {
		l.Info().
			Str("url", c.URL.String()).
			Msg("Trying to connect")
		c.mutexConn.Lock()
		c.conn, _, _, err = ws.DefaultDialer.Dial(context.Background(), c.URL.String())
		defer c.mutexConn.Unlock()
		if err == nil {
			l.Info().
				Str("url", c.URL.String()).
				Msg("Connection established")
			// resubscribing to registered events
			for _, e := range c.EventsSubscribed {
				c.EnqueueRequest(NewWebSocketRequest(e))
			}
			return
		}

		l.Error().
			Err(err).
			Msg("An error occurred during connection")

		time.Sleep(retryEvery)
	}
}

// NextMessageID returns the next usable message ID
func (c *WebSocketClient) NextMessageID() uint64 {
	atomic.AddUint64(&c.id, 1)
	return c.id
}

// Stop stops the web socket connection and free resources
func (c *WebSocketClient) Stop() {
	c.workerRequestsHandlerStop <- true

	// Cleaning conn and force workerDaemon to get out of the blocking reading func
	// Important to do that BEFORE sending the stop message
	// because when failing, a reconnection occurs, so connection HAS TO be nil before it happens
	// See workerDaemon func's error handling
	c.closeConn()
	c.workerDaemonStop <- true
}

// Start connects, authenticates and listens to Home Assistant WebSocket API
func (c *WebSocketClient) Start() error {
	c.mustConnect(2 * time.Second)

	defer func() {
		// registering default callbacks
		c.RegisterCallback("result", c.handleResult, model.HassResult{})
		c.RegisterCallback("auth_required", c.handleAuthRequired, model.HassResult{})
		c.RegisterCallback("auth_ok", c.handleAuthOK, model.HassResult{})
		c.RegisterCallback("auth_invalid", c.handleAuthInvalid, model.HassResult{})
	}()

	// 1 worker to send data to the server is enough
	defer app.RoutinesWG.Add(1)
	go func() {
		defer app.RoutinesWG.Done()
		c.workerRequestsHandler()
	}()

	// main thread handling communication with the server
	defer app.RoutinesWG.Add(1)
	go func() {
		defer app.RoutinesWG.Done()
		c.workerDaemon()
	}()

	return nil
}

func (c *WebSocketClient) recoverDisconnection() {
	l := logging.NewLogger("WebSocketClient.recoverDisconnection")
	if r := recover(); r != nil {
		l.Debug().
			Interface("recover", r).
			Msg("Panic recovered")
		c.mustConnect(2 * time.Second)
	}
}

// SubscribeEvents subscribes to Home Assistant event bus
func (c *WebSocketClient) SubscribeEvents(eventTypes ...string) {
	if c.EventsSubscribed == nil {
		c.EventsSubscribed = make(map[uint64]model.HassEventSubscription, 0)
	}

	for _, eventType := range eventTypes {
		sub := model.HassEventSubscription{
			ID:        c.NextMessageID(),
			EventType: eventType,
			Type:      "subscribe_events",
		}

		c.EventsSubscribed[sub.GetID()] = sub
		c.EnqueueRequest(NewWebSocketRequest(sub))
	}
}

// CallService is a generic function to call any service
// Deprecated: Might not work, to be debugged
func (c *WebSocketClient) CallService(entity model.HassEntity, service string) {
	l := logging.NewLogger("WebSocketClient.CallService")
	d := model.HassService{
		ID:      c.NextMessageID(),
		Type:    "call_service",
		Domain:  entity.Domain,
		Service: service,
		ServiceData: model.HassServiceData{
			EntityID: entity.GetEntityIDFullName(),
		},
	}

	l.Trace().
		EmbedObject(d).
		Msg("Calling service")
	c.EnqueueRequest(NewWebSocketRequest(d))
}

// EnqueueRequest queues a request to be sent to the server
func (c *WebSocketClient) EnqueueRequest(request *WebSocketRequest) {
	c.requestChannel <- request
}

// requeueRequest requeues a request after a while
func (c *WebSocketClient) requeueRequest(req *WebSocketRequest, after time.Duration) {
	app.RoutinesWG.Add(1)
	go func() {
		defer app.RoutinesWG.Done()
		time.Sleep(after)
		//req.Data = req.Data.Duplicate(c.NextMessageID())
		c.EnqueueRequest(req)
	}()
}

// workerRequestsHandler handles request from channel and effectively sends them to the server
func (c *WebSocketClient) workerRequestsHandler() {
	funcLogger := logging.NewLogger("WebSocketClient.workerRequestsHandler")

	running := true
	// Wait for message on the channel
	for running {
		select {
		case <-c.workerRequestsHandlerStop:
			funcLogger.Info().Msg("Stopping workerRequestsHandler routine")
			running = false
		case request := <-c.requestChannel:
			// Creating logger for this request
			l := funcLogger.With().
				Uint64("id", request.Data.GetID()).
				Str("type", request.Data.GetType()).
				Logger()

			if !c.authenticated && !strings.HasPrefix(request.Data.GetType(), "auth") {
				// not authenticated yet, requeue
				l.Debug().Msg("Not authenticated yet, requeuing request")
				c.requeueRequest(request, 1*time.Second)
				continue
			}

			if c.authenticated && !SimpleClientSingleton.CheckServerAPIHealth() {
				l.Warn().Msg("Server is unavailable, requeuing")
				c.requeueRequest(request, 2*time.Second)
				continue
			}

			// if c.conn is nil, something is wrong with the object (reloading config ?)
			if c.conn == nil {
				// ignoring message
				continue
			}

			l.Info().Msg("Processing request")
			data, _ := json.Marshal(request.Data)
			l.Trace().Msg("Sending request to the websocket server")
			err := wsutil.WriteServerMessage(c.conn, ws.OpText, data)
			if err != nil {
				l.Error().Err(err).Msg("Error sending request to the server, requeuing")
				c.EnqueueRequest(request)
			}

			// Track the request
			request.LastUpdateTime = time.Now()
			c.requestsTracker.InProgress(request.Data.GetID(), request)
		}
	} // for running

	funcLogger.Info().Msg("workerRequestsHandler stopped")
}

func (c *WebSocketClient) workerDaemon() {
	l := logging.NewLogger("WebSocketClient.workerDaemon")

	defer c.recoverDisconnection() // calling that when wsutil.ReadServerData panics
	running := true
	for running {
		select {
		case <-c.workerDaemonStop:
			l.Info().Msg("Stopping workerDaemon routine")
			running = false
			continue
		default: // Avoid vscode complaining about unreachable code below
		}

		var msg struct {
			Type string `json:"type"`
		}

		// ReadServerData is a blocking func
		// As such, the only way to force it to return is to close the connection and force it to nil
		// A mutex is used so that when stop is called from another go routine, only one go routine
		// can change conn object at a time.
		// Otherwise, when testing if conn == nil, it might or might not be the case yet...
		l.Trace().Msg("Waiting for a message from the server...")
		recv, _, err := wsutil.ReadServerData(c.conn)
		if err != nil {
			c.mutexConn.Lock()
			connIsNil := c.conn == nil
			c.mutexConn.Unlock()
			if connIsNil { // Killed via stop function using a nasty workaround: closing conn and setting it to nil...
				l.Trace().Msg("Stop func has been called")
				continue
			}

			l.Error().Err(err).Msg("Error reading from server")
			c.mustConnect(2 * time.Second)
			// abort current loop
			continue
		}

		l.Trace().
			Bytes("data", recv).
			Msg("Message received from the server")

		err = json.Unmarshal(recv, &msg)
		if err != nil {
			l.Error().Err(err).Msg("Error receiving message from server")
			continue
		}

		// Calling callback if registered
		if cb, ok := c.callbacks[msg.Type]; ok {
			// Creating obj from the one passed with RegisterCallback and converting it to model.HassAPIObject
			// Calling callback with this interface
			obj := reflect.New(reflect.TypeOf(cb.ConcreteType)).Interface().(model.HassAPIObject)
			if cb.F != nil {
				if err := json.Unmarshal(recv, &obj); err != nil {
					l.Error().Err(err).Msg("Unable to unmarshal data")
				} else {
					app.RoutinesWG.Add(1)
					go func() {
						defer app.RoutinesWG.Done()
						cb.F(obj)
					}()
				}
			}
		} else {
			// No handler configured
			l.Warn().
				Str("message", string(recv)).
				Msg("No handler defined")
		}
	}

	l.Info().Msg("workerDaemon stopped")
}

// Authenticated returns true if already authenticated, false otherwise
func (c *WebSocketClient) Authenticated() bool {
	return c.authenticated
}

func (c *WebSocketClient) closeConn() {
	c.mutexConn.Lock()
	defer c.mutexConn.Unlock()
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
}

// handleResult handles result to a previously sent request and update request object accordingly
func (c *WebSocketClient) handleResult(data model.HassAPIObject) {
	l := logging.NewLogger("WebSocketClient.handleResult")

	result := data.(*model.HassResult)
	if result.Success {
		l.Debug().
			Uint64("id", result.GetID()).
			Msg("Success result received for request")
	}

	req := c.requestsTracker.Done(result.GetID())
	if !result.Success {
		after := 3 * time.Second
		if result.Error.Code == ErrorCodeIDReuse || result.Error.Code == ErrorInvalidFormat {
			after = 10 * time.Millisecond // retry sooner than later
			req.Data = req.Data.Duplicate(c.NextMessageID())
		}

		l.Warn().
			Uint64("id", result.GetID()).
			Str("error_code", result.Error.Code).
			Str("error_message", result.Error.Message).
			Msgf("Failed result received for request, requeuing in %s", after.String())

		c.requeueRequest(req, after)
	}
}

func (c *WebSocketClient) handleAuthRequired(data model.HassAPIObject) {
	l := logging.NewLogger("WebSocketClient.handleAuthRequired")
	l.Info().
		Str("type", data.GetType()).
		Msgf("Message received from server")
	c.EnqueueRequest(NewWebSocketRequest(model.NewHassAuthentication(c.HassConfig.Token)))
}

func (c *WebSocketClient) handleAuthOK(data model.HassAPIObject) {
	l := logging.NewLogger("WebSocketClient.handleAuthOK")
	l.Info().
		Str("type", data.GetType()).
		Msgf("Message received from server")
	c.authenticated = true
}

func (c *WebSocketClient) handleAuthInvalid(data model.HassAPIObject) {
	l := logging.NewLogger("WebSocketClient.handleAuthInvalid")
	result := data.(*model.HassResult)
	l.Error().
		Err(fmt.Errorf(result.Message)).
		Str("type", result.GetType()).
		Msgf("Message received from server, cannot continue")
	c.authenticated = false
	c.Stop()
}
