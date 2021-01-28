package httpclient

import (
	"context"
	"encoding/json"
	"net"
	"reflect"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
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
	conn             net.Conn
	callbacks        map[string]callback
	EventsSubscribed map[uint64]model.HassEventSubscription
	// id to use for the next request
	id uint64
	// requestChannel is used to share WebSocketRequest objects between go routines
	requestChannel chan *WebSocketRequest
	// requestsTracker keeps track of requests sent to the server waiting for a result
	requestsTracker WebSocketRequestsTracker
	authenticated   bool
}

// NewWebSocketClient returns a new NewWebSocketClient initialized
func NewWebSocketClient(config model.HassConfig) *WebSocketClient {
	return &WebSocketClient{
		HassConfig:     config,
		requestChannel: make(chan *WebSocketRequest, 100),
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
	var err error

	// forcing connection to close
	if c.conn != nil {
		c.conn.Close()
	}
	c.authenticated = false
	atomic.StoreUint64(&c.id, 0)

	for {
		logging.Info("mustConnect").
			Str("url", c.URL.String()).
			Msg("Trying to connect")
		c.conn, _, _, err = ws.DefaultDialer.Dial(context.Background(), c.URL.String())
		if err == nil {
			// resubscribing to registered events
			for _, e := range c.EventsSubscribed {
				c.EnqueueRequest(NewWebSocketRequest(e))
			}
			return
		}

		logging.Error("mustConnect").
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
	if c.conn != nil {
		c.conn.Close()
	}
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
	go c.workerRequestsHandler()

	// main thread handling communication with the server
	go c.workerDaemon()

	return nil
}

func (c *WebSocketClient) handleDisconnection() {
	if r := recover(); r != nil {
		logging.Debug("handleDisconnection").
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
	d := model.HassService{
		ID:      c.NextMessageID(),
		Type:    "call_service",
		Domain:  entity.Domain,
		Service: service,
		ServiceData: model.HassServiceData{
			EntityID: entity.GetEntityIDFullName(),
		},
	}

	logging.Trace("CallService").
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
	go func() {
		time.Sleep(after)
		//req.Data = req.Data.Duplicate(c.NextMessageID())
		c.EnqueueRequest(req)
	}()
}

// workerRequestsHandler handles request from channel and effectively sends them to the server
func (c *WebSocketClient) workerRequestsHandler() {
	// Wait for message on the channel
	for request := range c.requestChannel {
		if !SimpleClientSingleton.CheckServerAPIHealth() {
			logging.Warn("WebSocketClient.workerRequestsHandler").
				Uint64("id", request.Data.GetID()).
				Str("type", request.Data.GetType()).
				Msg("Server is unavailable, requeuing")
			c.requeueRequest(request, 2*time.Second)
			continue
		}

		if !c.authenticated && !strings.HasPrefix(request.Data.GetType(), "auth") {
			// not authenticated yet, requeue
			logging.Info("WebSocketClient.workerRequestsHandler").
				Uint64("id", request.Data.GetID()).
				Str("type", request.Data.GetType()).
				Msg("Not authenticated yet, requeuing request")
			c.requeueRequest(request, 1*time.Second)
			continue
		}

		logging.Info("WebSocketClient.workerRequestsHandler").
			Uint64("id", request.Data.GetID()).
			Str("type", request.Data.GetType()).
			Msg("Processing request")

		data, _ := json.Marshal(request.Data)
		logging.Trace("WebSocketClient.workerRequestsHandler").
			Uint64("id", request.Data.GetID()).
			Str("type", request.Data.GetType()).
			Msg("Sending request to the websocket server")
		err := wsutil.WriteServerMessage(c.conn, ws.OpText, data)
		if err != nil {
			logging.Error("WebSocketClient.workerRequestsHandler").Err(err).Msg("Error sending request to the server, requeuing")
			c.EnqueueRequest(request)
		}

		// Track the request
		request.LastUpdateTime = time.Now()
		c.requestsTracker.InProgress(request.Data.GetID(), request)
	}
}

func (c *WebSocketClient) workerDaemon() {
	defer c.handleDisconnection() // calling that when panicking
	for {
		var msg struct {
			Type string `json:"type"`
		}

		recv, _, err := wsutil.ReadServerData(c.conn) // this is a blocking func
		if err != nil {
			logging.Error("WebSocketClient.workerDaemon").Err(err).Msg("Error reading from server")
			c.conn.Close()
			c.conn = nil
			c.mustConnect(2 * time.Second)
			defer c.conn.Close()
			// abort current loop
			continue
		}

		logging.Trace("WebSocketClient.workerDaemon").
			Bytes("data", recv).
			Msg("Message received from the server")

		err = json.Unmarshal(recv, &msg)
		if err != nil {
			logging.Error("WebSocketClient.workerDaemon").Err(err).Msg("Error receiving message from server")
			continue
		}

		// Calling callback if registered
		if cb, ok := c.callbacks[msg.Type]; ok {
			// Creating obj from the one passed with RegisterCallback and converting it to model.HassAPIObject
			// Calling callback with this interface
			obj := reflect.New(reflect.TypeOf(cb.ConcreteType)).Interface().(model.HassAPIObject)
			if cb.F != nil {
				if err := json.Unmarshal(recv, &obj); err != nil {
					logging.Error("WebSocketClient.workerDaemon").Err(err).Msg("Unable to unmarshal data")
				} else {
					go cb.F(obj)
				}
			}
		} else {
			// No handler configured
			logging.Warn("WebSocketClient.workerDaemon").
				Str("message", string(recv)).
				Msg("No handler defined")
		}
	}
}

// Authenticated returns true if already authenticated, false otherwise
func (c *WebSocketClient) Authenticated() bool {
	return c.authenticated
}

// handleResult handles result to a previously sent request and update request object accordingly
func (c *WebSocketClient) handleResult(data model.HassAPIObject) {
	result := data.(*model.HassResult)
	if result.Success {
		logging.Debug("WebSocketClient.handleResult").
			Uint64("id", result.GetID()).
			Msg("Success result received for request")
	} else {
		logging.Warn("WebSocketClient.handleResult").
			Uint64("id", result.GetID()).
			Str("error_code", result.Error.Code).
			Str("error_message", result.Error.Message).
			Msg("Failed result received for request")
	}

	req := c.requestsTracker.Done(result.GetID())
	if !result.Success {
		after := 3 * time.Second

		if result.Error.Code == ErrorCodeIDReuse || result.Error.Code == ErrorInvalidFormat {
			after = 10 * time.Millisecond // retry sooner than later
			req.Data = req.Data.Duplicate(c.NextMessageID())
		}

		c.requeueRequest(req, after)
	}
}

func (c *WebSocketClient) handleAuthRequired(data model.HassAPIObject) {
	logging.Info("WebSocketClient.handleAuthRequired").
		Str("type", data.GetType()).
		Msgf("Message received from server")
	c.EnqueueRequest(NewWebSocketRequest(model.NewHassAuthentication(c.HassConfig.Token)))
}

func (c *WebSocketClient) handleAuthOK(data model.HassAPIObject) {
	logging.Info("WebSocketClient.handleAuthOK").
		Str("type", data.GetType()).
		Msgf("Message received from server")
	c.authenticated = true
}

func (c *WebSocketClient) handleAuthInvalid(data model.HassAPIObject) {
	result := data.(*model.HassResult)
	logging.Info("WebSocketClient.handleAuthInvalid").
		Str("reason", result.Message).
		Str("type", result.GetType()).
		Msgf("Message received from server")
	c.authenticated = false
}
