package model

const (
	// StateON is the string used when state is ON
	StateON = "on"
	// StateOFF is the string used when state is OFF
	StateOFF = "off"
)

// HassState represents a Home Assistant entity
type HassState struct {
	EntityID    string                 `json:"entity_id"`
	LastChanged string                 `json:"last_changed"`
	State       string                 `json:"state"`
	Attributes  map[string]interface{} `json:"attributes"`
	LastUpdated string                 `json:"last_updated"`
	Context     HassContext            `json:"context"`
}
