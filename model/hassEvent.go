package model

import "github.com/rs/zerolog"

var (
	_ HassAPIObject              = (*HassEvent)(nil)
	_ zerolog.LogObjectMarshaler = (*HassEvent)(nil)
)

// HassEvent represents a Home Assistant event
type HassEvent struct {
	ID    uint64           `json:"id"`
	Type  string           `json:"type"`
	Event HassEventContent `json:"event,omitempty"`
}

// HassEventContent godoc
type HassEventContent struct {
	EventType string        `json:"event_type"`
	Data      HassEventData `json:"data"`
	Origin    string        `json:"origin"`
	TimeFired string        `json:"time_fired"`
	Context   HassContext   `json:"context"`
}

// HassEventData is the data of the event
// All fields might not be filled, it depends on the event EventType
type HassEventData struct {
	EntityID   string    `json:"entity_id"`
	SourceName string    `json:"source_name"`
	Type       string    `json:"type"`
	Key        string    `json:"key"`
	OldState   HassState `json:"old_state"`
	NewState   HassState `json:"new_state"`
}

// GetID godoc
func (e HassEvent) GetID() uint64 {
	return e.ID
}

// GetType godoc
func (e HassEvent) GetType() string {
	return e.Type
}

// Duplicate godoc
func (e HassEvent) Duplicate(id uint64) HassAPIObject {
	dup := e
	dup.ID = id
	return dup
}

// SameState returns true if old state and new state matches
func (e HassEvent) SameState() bool {
	return e.Event.Data.OldState.State == e.Event.Data.NewState.State
}

// OppositeState returns true if old state is the opposite of new state
// An empty state is considered 'off'
func (e HassEvent) OppositeState() bool {
	oldState := e.Event.Data.OldState.State
	newState := e.Event.Data.NewState.State
	if oldState == "" {
		oldState = StateOFF
	}
	if newState == "" {
		newState = StateOFF
	}

	return oldState != newState
}

// MarshalZerologObject godoc
func (e HassEvent) MarshalZerologObject(event *zerolog.Event) {
	event.
		Uint64("id", e.ID).
		Str("type", e.Type).
		Str("event.event_type", e.Event.EventType).
		Str("event.data.entity_id", e.Event.Data.EntityID).
		Str("event.data.old_state", e.Event.Data.OldState.State).
		Str("event.data.new_state", e.Event.Data.NewState.State)
}
