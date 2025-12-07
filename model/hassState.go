package model

import (
	"reflect"
	"strings"
	"time"
)

const (
	// StateON is the string used when state is ON
	StateON = "on"
	// StateOFF is the string used when state is OFF
	StateOFF = "off"
)

// HassState represents a Home Assistant entity's state
type HassState struct {
	EntityID     string                 `json:"entity_id"`
	LastChanged  time.Time              `json:"last_changed"`
	LastUpdated  time.Time              `json:"last_updated"`
	LastReported time.Time              `json:"last_reported"`
	State        string                 `json:"state"`
	Attributes   map[string]interface{} `json:"attributes"`
	Context      HassContext            `json:"context"`
}

func (s HassState) String() string {
	return strings.ToLower(s.State)
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

// GetAttrAsBool returns the attribute as bool
// If attribute does not exist, return false
// If attribute is not a bool, return false
func (s HassState) GetAttrAsBool(attr string) bool {
	a, ok := s.Attributes[attr]
	if !ok {
		return false
	}

	if reflect.TypeOf(a).Kind() == reflect.Bool {
		return a.(bool)
	}

	return false
}
