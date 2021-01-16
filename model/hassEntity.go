package model

// HassEntity represents a Home Assistant entity
type HassEntity struct {
	EntityID string
	Domain   string
	State    HassState
}
