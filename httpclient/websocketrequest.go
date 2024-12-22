package httpclient

import (
	"time"

	"github.com/nmaupu/gotomation/model"
	"github.com/rs/zerolog"
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

// MarshalZerologObject godoc
func (r WebSocketRequest) MarshalZerologObject(event *zerolog.Event) {
	event.
		Uint64("data_id", r.Data.GetID()).
		Str("data_type", r.Data.GetType()).
		Time("creation_time", r.CreationTime).
		Time("last_update_time", r.LastUpdateTime)
}
