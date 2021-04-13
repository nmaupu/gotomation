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
	"github.com/nmaupu/gotomation/routines"
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
type WebSocketClient interface {
	routines.Runnable
	RegisterCallback(hassType string, f ResponseHandlerSignature, concreteType model.HassAPIObject)
	SubscribeEvents(eventTypes ...string)
}

type webSocketClient struct {
	// id to use for the next request - has to be declared first
	// see https://golang.org/pkg/sync/atomic/#pkg-note-BUG
	// see https://github.com/census-instrumentation/opencensus-go/issues/587
	id uint64

	model.HassConfig
	mutexConn             sync.Mutex
	conn                  net.Conn
	callbacks             map[string]callback
	mutexEventsSubscribed sync.Mutex
	EventsSubscribed      map[uint64]model.HassEventSubscription

	// requestChannel is used to share WebSocketRequest objects between go routines
	requestChannel chan *WebSocketRequest
	// workerRequestsHandlerStop is used to stop workerRequestsHandler routine
	workerRequestsHandlerStop chan bool
	// workerDaemonStop is used to stop workerDaemon routine
	workerDaemonStop chan bool

	// requestsTracker keeps track of requests sent to the server waiting for a result
	requestsTracker WebSocketRequestsTracker

	mutexAuthenticated sync.Mutex
	authenticated      bool

	mutexConnected sync.Mutex
	connected      bool

	// started indicates whether or not runnable is started
	mutexStopStart sync.Mutex
	started        bool
}

// NewWebSocketClient returns a new NewWebSocketClient initialized
func NewWebSocketClient(config model.HassConfig) WebSocketClient {
	return &webSocketClient{
		HassConfig:     config,
		requestChannel: make(chan *WebSocketRequest, 10),
		// important: setting size of chan bools to 1 to avoid being blocked until read when sending the stop message
		workerRequestsHandlerStop: make(chan bool, 1),
		workerDaemonStop:          make(chan bool, 1),
	}
}

// RegisterCallback registers a new callback given its type
func (c *webSocketClient) RegisterCallback(hassType string, f ResponseHandlerSignature, concreteType model.HassAPIObject) {
	if c.callbacks == nil {
		c.callbacks = make(map[string]callback, 0)
	}

	c.callbacks[hassType] = callback{
		F:            f,
		ConcreteType: concreteType,
	}
}

// DeregisterCallback deregisters a callback given its type
func (c *webSocketClient) DeregisterCallback(hassType string) {
	if c.callbacks != nil {
		delete(c.callbacks, hassType)
	}
}

func (c *webSocketClient) mustConnect(retryEvery time.Duration) {
	l := logging.NewLogger("WebSocketClient.mustConnect")
	var err error

	// forcing connection to close
	c.setConnected(false)
	c.closeConn()
	c.SetAuthenticated(false)
	atomic.StoreUint64(&c.id, 0)

	for {
		l.Info().
			Str("url", c.URL.String()).
			Msg("Trying to connect")
		c.mutexConn.Lock()
		c.conn, _, _, err = ws.DefaultDialer.Dial(context.Background(), c.URL.String())
		c.mutexConn.Unlock()
		if err == nil {
			c.setConnected(true)
			l.Info().
				Str("url", c.URL.String()).
				Msg("Connection established")
			// resubscribing to registered events
			c.mutexEventsSubscribed.Lock()
			defer c.mutexEventsSubscribed.Unlock()
			for _, e := range c.EventsSubscribed {
				if !c.requestsTracker.IsInProgress(e.GetID()) {
					c.EnqueueRequest(NewWebSocketRequest(e))
				}
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
func (c *webSocketClient) NextMessageID() uint64 {
	atomic.AddUint64(&c.id, 1)
	return c.id
}

// Stop stops the web socket connection and free resources
func (c *webSocketClient) Stop() {
	c.mutexStopStart.Lock()
	defer c.mutexStopStart.Unlock()
	if !c.started {
		return
	}

	c.workerRequestsHandlerStop <- true

	// Cleaning conn and force workerDaemon to get out of the blocking reading func
	// Important to do that BEFORE sending the stop message
	// because when failing, a reconnection occurs, so connection HAS TO be nil before it happens
	// See workerDaemon func's error handling
	c.closeConn()
	c.workerDaemonStop <- true
	c.started = false
}

// Start connects, authenticates and listens to Home Assistant WebSocket API
func (c *webSocketClient) Start() error {
	c.mutexStopStart.Lock()
	defer c.mutexStopStart.Unlock()

	if c.started {
		return nil
	}

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

	c.started = true
	return nil
}

func (c *webSocketClient) IsStarted() bool {
	c.mutexStopStart.Lock()
	defer c.mutexStopStart.Unlock()
	return c.started
}

func (c *webSocketClient) recoverDisconnection() {
	l := logging.NewLogger("WebSocketClient.recoverDisconnection")
	if r := recover(); r != nil {
		l.Debug().
			Interface("recover", r).
			Msg("Panic recovered")
		c.mustConnect(2 * time.Second)
	}
}

// SubscribeEvents subscribes to Home Assistant event bus
func (c *webSocketClient) SubscribeEvents(eventTypes ...string) {
	c.mutexEventsSubscribed.Lock()
	defer c.mutexEventsSubscribed.Unlock()

	l := logging.NewLogger("webSocketClient.SubscribeEvents")
	l.Trace().Strs("event_types", eventTypes).Msg("Subscribing to events")

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
		if c.Connected() { // Don't enqueue if not connected because mustConnect will do it
			c.EnqueueRequest(NewWebSocketRequest(sub))
		}
	}
}

// CallService is a generic function to call any service
// TODO Deprecated: Might not work, to be debugged
func (c *webSocketClient) CallService(entity model.HassEntity, service string) {
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
func (c *webSocketClient) EnqueueRequest(request *WebSocketRequest) {
	c.requestChannel <- request
}

// requeueRequest requeues a request after a while
func (c *webSocketClient) requeueRequest(req *WebSocketRequest, after time.Duration) {
	app.RoutinesWG.Add(1)
	go func() {
		defer app.RoutinesWG.Done()
		time.Sleep(after)
		//req.Data = req.Data.Duplicate(c.NextMessageID())
		c.EnqueueRequest(req)
	}()
}

// workerRequestsHandler handles requests from channel and effectively sends them to the server
func (c *webSocketClient) workerRequestsHandler() {
	funcLogger := logging.NewLogger("WebSocketClient.workerRequestsHandler")

loop:
	for {
		select {
		case <-c.workerRequestsHandlerStop:
			funcLogger.Trace().Msg("Stopping workerRequestsHandler routine")
			break loop
		case request := <-c.requestChannel:
			// Creating logger for this request
			l := funcLogger.With().
				Uint64("id", request.Data.GetID()).
				Str("type", request.Data.GetType()).
				Logger()

			// Track the request
			request.LastUpdateTime = time.Now()
			c.requestsTracker.InProgress(request)

			if !GetSimpleClient().CheckServerAPIHealth() {
				l.Warn().Msg("Server is unavailable, requeuing")
				c.requeueRequest(request, 2*time.Second)
				continue
			}

			if !c.Authenticated() && !strings.HasPrefix(request.Data.GetType(), "auth") {
				// not authenticated yet, requeue
				l.Debug().Msg("Not authenticated yet, requeuing request")
				c.requeueRequest(request, 1*time.Second)
				continue
			}

			// if c.conn is nil, something is wrong with the object (reloading config ?)
			if c.conn == nil {
				// ignoring message
				continue
			}

			l.Info().Msg("Processing request")
			data, _ := json.Marshal(request.Data)
			l.Trace().Bytes("data", data).Msg("Sending request to the websocket server")
			err := wsutil.WriteServerMessage(c.conn, ws.OpText, data)
			if err != nil {
				l.Error().Err(err).Msg("Error sending request to the server, requeuing")
				c.EnqueueRequest(request)
				continue
			}
		}
	} // for running

	funcLogger.Trace().Msg("workerRequestsHandler stopped")
}

func (c *webSocketClient) workerDaemon() {
	l := logging.NewLogger("WebSocketClient.workerDaemon")

	defer c.recoverDisconnection() // calling that when wsutil.ReadServerData panics
loop:
	for {
		select {
		case <-c.workerDaemonStop:
			l.Trace().Msg("Stopping workerDaemon routine")
			break loop
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

	l.Trace().Msg("workerDaemon stopped")
}

func (c *webSocketClient) closeConn() {
	c.mutexConn.Lock()
	defer c.mutexConn.Unlock()
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
}

// handleResult handles result to a previously sent request and update request object accordingly
func (c *webSocketClient) handleResult(data model.HassAPIObject) {
	l := logging.NewLogger("WebSocketClient.handleResult")

	result := data.(*model.HassResult)
	req := c.requestsTracker.Done(result.GetID())

	if result.Success {
		l.Debug().
			Uint64("id", result.GetID()).
			Msg("Success result received for request")
		return
	}

	if req == nil {
		l.Warn().Msg("Result is failed but request is nil, cannot requeue")
		return
	}

	// Not successful
	after := 3 * time.Second
	if result.Error.Code == ErrorCodeIDReuse || result.Error.Code == ErrorInvalidFormat {
		after = 10 * time.Millisecond // retry sooner than later
		req.Data = req.Data.Duplicate(c.NextMessageID())
	}

	l.Debug().
		Uint64("id", result.GetID()).
		Str("error_code", result.Error.Code).
		Str("error_message", result.Error.Message).
		Msgf("Failed result received for request, requeuing in %s", after.String())

	c.requeueRequest(req, after)
}

func (c *webSocketClient) handleAuthRequired(data model.HassAPIObject) {
	l := logging.NewLogger("WebSocketClient.handleAuthRequired")
	l.Info().
		Str("type", data.GetType()).
		Msgf("Message received from server")
	c.EnqueueRequest(NewWebSocketRequest(model.NewHassAuthentication(c.HassConfig.Token)))
}

func (c *webSocketClient) handleAuthOK(data model.HassAPIObject) {
	l := logging.NewLogger("WebSocketClient.handleAuthOK")
	l.Info().
		Str("type", data.GetType()).
		Msgf("Message received from server")
	c.SetAuthenticated(true)
}

func (c *webSocketClient) handleAuthInvalid(data model.HassAPIObject) {
	l := logging.NewLogger("WebSocketClient.handleAuthInvalid")
	result := data.(*model.HassResult)
	l.Error().
		Err(fmt.Errorf(result.Message)).
		Str("type", result.GetType()).
		Msgf("Message received from server, cannot continue")
	c.SetAuthenticated(false)
	c.Stop()
}

// Authenticated returns true if already authenticated, false otherwise
func (c *webSocketClient) Authenticated() bool {
	c.mutexAuthenticated.Lock()
	defer c.mutexAuthenticated.Unlock()
	return c.authenticated
}

func (c *webSocketClient) SetAuthenticated(b bool) {
	c.mutexAuthenticated.Lock()
	defer c.mutexAuthenticated.Unlock()
	c.authenticated = b
}

func (c *webSocketClient) Connected() bool {
	c.mutexConnected.Lock()
	defer c.mutexConnected.Unlock()
	return c.connected
}

func (c *webSocketClient) setConnected(b bool) {
	c.mutexConnected.Lock()
	defer c.mutexConnected.Unlock()
	c.connected = b
}

// GetName returns the name of this runnable object
func (c *webSocketClient) GetName() string {
	return "WebSocketClient"
}
