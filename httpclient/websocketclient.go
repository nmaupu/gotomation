package httpclient

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"reflect"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
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
		log.Println("Connecting to", c.URL.String())
		c.conn, _, _, err = ws.DefaultDialer.Dial(context.Background(), c.URL.String())
		if err == nil {
			// resubscribing to registered events
			for _, e := range c.EventsSubscribed {
				c.EnqueueRequest(NewWebSocketRequest(e))
			}
			return
		}

		log.Print(err)

		time.Sleep(retryEvery)
	}
}

// NextMessageID returns the next usable message ID
func (c *WebSocketClient) NextMessageID() uint64 {
	atomic.AddUint64(&c.id, 1)
	return c.id
}

// Close closes the communication with the server
func (c *WebSocketClient) Close() {
	c.conn.Close()
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
		log.Printf("Panic recovered, r=%v", r)
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
func (c *WebSocketClient) CallService(domain string, service string, entity string) {
	d := model.HassService{
		ID:      c.NextMessageID(),
		Type:    "call_service",
		Domain:  domain,
		Service: service,
		ServiceData: model.HassServiceData{
			EntityID: entity,
		},
	}

	c.EnqueueRequest(NewWebSocketRequest(d))
}

// LightTurnOn turns on a given light
func (c *WebSocketClient) LightTurnOn(entity string) {
	c.LightSet("turn_on", entity)
}

// LightTurnOff turns off a given light
func (c *WebSocketClient) LightTurnOff(entity string) {
	c.LightSet("turn_off", entity)
}

// LightToggle toggles a given light
func (c *WebSocketClient) LightToggle(entity string) {
	c.LightSet("toggle", entity)
}

// LightSet calls turn_on/turn_off/toggle on a given light
func (c *WebSocketClient) LightSet(service string, entity string) {
	c.CallService("light", service, entity)
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
			log.Printf("Server is unavailable, requeuing id=%d, type=%s", request.Data.GetID(), request.Data.GetType())
			c.requeueRequest(request, 2*time.Second)
			continue
		}

		if !c.authenticated && !strings.HasPrefix(request.Data.GetType(), "auth") {
			// not authenticated yet, requeue
			log.Printf("Not authenticated yet, requeue request id=%d, type=%s", request.Data.GetID(), request.Data.GetType())
			c.requeueRequest(request, 1*time.Second)
			continue
		}

		log.Printf("Processing request id=%d, type=%s", request.Data.GetID(), request.Data.GetType())

		data, _ := json.Marshal(request.Data)
		log.Printf("Sending request to the WebSocket server, id=%d, type=%s", request.Data.GetID(), request.Data.GetType())
		err := wsutil.WriteServerMessage(c.conn, ws.OpText, data)
		if err != nil {
			log.Printf("Error sending request to the server, requeuing. err=%v", err)
			c.EnqueueRequest(request)
		}

		// Track the request
		request.LastUpdateTime = time.Now()
		c.requestsTracker.InProgress(request.Data.GetID(), request)
	}
}

func (c *WebSocketClient) workerDaemon() {
	for {
		var msg struct {
			Type string `json:"type"`
		}

		//log.Printf("Waiting for a message from the server...")
		defer c.handleDisconnection()                 // calling that when panicking
		recv, _, err := wsutil.ReadServerData(c.conn) // ! this is a blocking func
		if err != nil {
			log.Printf("Error reading from server, err=%+v", err)
			c.conn.Close()
			c.conn = nil
			c.mustConnect(2 * time.Second)
			defer c.conn.Close()
			// abort current loop
			continue
		}

		err = json.Unmarshal(recv, &msg)
		if err != nil {
			log.Printf("Error receiving message from server, err=%v", err)
			continue
		}

		// Calling callback if registered
		if cb, ok := c.callbacks[msg.Type]; ok {
			// Creating obj from the one passed with RegisterCallback and converting it to model.HassAPIObject
			// Calling callback with this interface
			obj := reflect.New(reflect.TypeOf(cb.ConcreteType)).Interface().(model.HassAPIObject)
			if cb.F != nil {
				if err := json.Unmarshal(recv, &obj); err != nil {
					log.Printf("Unable to unmarshal data, err=%v", err)
				} else {
					cb.F(obj)
				}
			}
		} else {
			// No handler configured
			log.Printf("No handler for message: %+v", string(recv))
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
		log.Printf("Result received for request id %d, success=%t", result.GetID(), result.Success)
	} else {
		log.Printf("Result received for request id %d, success=%t, errorCode=%s, errorMessage=%s", result.GetID(), result.Success, result.Error.Code, result.Error.Message)
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
	log.Printf("Received %s, authenticating.", data.GetType())
	c.EnqueueRequest(NewWebSocketRequest(model.NewHassAuthentication(c.HassConfig.Token)))
}

func (c *WebSocketClient) handleAuthOK(data model.HassAPIObject) {
	log.Printf("Received %s", data.GetType())
	c.authenticated = true
}

func (c *WebSocketClient) handleAuthInvalid(data model.HassAPIObject) {
	result := data.(*model.HassResult)
	log.Printf("Received %s, reason=%s", result.GetType(), result.Message)
	c.authenticated = false
}
