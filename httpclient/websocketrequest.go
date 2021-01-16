package httpclient

import (
	"time"

	"github.com/nmaupu/gotomation/model"
)

// WebSocketRequest is a request made from this client
type WebSocketRequest struct {
	Data           model.HassAPIObject
	CreationTime   time.Time
	LastUpdateTime time.Time
}

// NewWebSocketRequest creates a new WebSocketRequest
func NewWebSocketRequest(data model.HassAPIObject) *WebSocketRequest {
	return &WebSocketRequest{
		Data:         data,
		CreationTime: time.Now(),
	}
}
