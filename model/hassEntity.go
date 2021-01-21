package model

import (
	"fmt"
	"strings"
)

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

// NewHassEntity returns a new HassEntity from a full name such as light.living
// where Domain: light and EntityID: living
func NewHassEntity(entityID string) HassEntity {
	vals := strings.Split(entityID, ".")

	if len(vals) < 2 {
		return HassEntity{}
	}

	return HassEntity{
		Domain:   vals[0],
		EntityID: strings.Join(vals[1:], "."),
	}
}

// Equals returns true if both entities are equals (same domain and entity_id), false otherwise
func (e HassEntity) Equals(entity HassEntity) bool {
	return e.Domain == entity.Domain && e.EntityID == entity.EntityID
}

// IsContained returns true if e is in entities, false otherwise
func (e HassEntity) IsContained(entities []HassEntity) bool {
	for _, entity := range entities {
		if entity.Equals(e) {
			return true
		}
	}

	return false
}
