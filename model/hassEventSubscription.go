package model

var (
	_ HassAPIObject = (*HassEventSubscription)(nil)
)

// HassEventSubscription is the data sent to subscribe to events
type HassEventSubscription struct {
	ID        uint64 `json:"id"`
	Type      string `json:"type"`
	EventType string `json:"event_type"`
}

// GetID godoc
func (s HassEventSubscription) GetID() uint64 {
	return s.ID
}

// GetType godoc
func (s HassEventSubscription) GetType() string {
	return s.Type
}

// Duplicate godoc
func (s HassEventSubscription) Duplicate(id uint64) HassAPIObject {
	dup := s
	dup.ID = id
	return dup
}
