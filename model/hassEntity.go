package model

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/nmaupu/gotomation/logging"
	"github.com/rs/zerolog"
)

var (
	_ zerolog.LogObjectMarshaler = (*HassEntity)(nil)
)

// HassEntity represents a Home Assistant entity
type HassEntity struct {
	EntityID string
	Domain   string
	State    HassState
}

// GetEntityIDFullName return the entity_id in the form domain.entity_id
func (e HassEntity) GetEntityIDFullName() string {
	if e.Domain == "" && e.EntityID == "" {
		return ""
	}

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
// regexp are supported for entity_id
func (e HassEntity) Equals(entity HassEntity) bool {
	l := logging.NewLogger("HassEntity.Equals")

	if e.Domain != entity.Domain {
		return false
	}

	re, err := regexp.Compile(entity.EntityID)
	if err != nil {
		return false
	}

	res := re.MatchString(e.EntityID)
	l.Trace().
		Str("entity", e.GetEntityIDFullName()).
		Str("candidate", fmt.Sprintf(entity.GetEntityIDFullName())).
		Bool("response", res).
		Send()

	return res
}

// IsContained returns true if e is in entities, false otherwise
func (e HassEntity) IsContained(entities []HassEntity) bool {
	for _, entity := range entities {
		if e.Equals(entity) {
			return true
		}
	}

	return false
}

// MarshalZerologObject godoc
func (e HassEntity) MarshalZerologObject(event *zerolog.Event) {
	event.
		Str("entity_id", e.EntityID).
		Str("domain", e.Domain).
		Str("state", e.State.State)
}
