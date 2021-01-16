package model

import "fmt"

// HassEntity represents a Home Assistant entity
type HassEntity struct {
	EntityID string
	Domain   string
	State    HassState
}

// GetEntityIDFullName return the entity_id in the form domain.entity_id
func (e HassEntity) GetEntityIDFullName() string {
	return fmt.Sprintf("%s.%s", e.Domain, e.EntityID)
}
