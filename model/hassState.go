package model

import "strings"

const (
	// StateON is the string used when state is ON
	StateON = "on"
	// StateOFF is the string used when state is OFF
	StateOFF = "off"
)

// HassState represents a Home Assistant entity's state
type HassState struct {
	EntityID    string                 `json:"entity_id"`
	LastChanged string                 `json:"last_changed"`
	State       string                 `json:"state"`
	Attributes  map[string]interface{} `json:"attributes"`
	LastUpdated string                 `json:"last_updated"`
	Context     HassContext            `json:"context"`
}

// IsON returns true if State is set to 'on'
// Is state is not set, state is considered OFF
func (s HassState) IsON() bool {
	return strings.ToLower(s.State) == strings.ToLower(StateON)
}

// IsOFF returns true if State is set to 'off'
// Is state is not set, state is considered OFF
func (s HassState) IsOFF() bool {
	return !s.IsON()
}
