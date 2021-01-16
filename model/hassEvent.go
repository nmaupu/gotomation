package model

var (
	_ HassAPIObject = (*HassEvent)(nil)
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

// HassEventData godoc
type HassEventData struct {
	EntityID string    `json:"entity_id"`
	OldState HassState `json:"old_state"`
	NewState HassState `json:"new_state"`
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
