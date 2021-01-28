package model

import "github.com/rs/zerolog"

var (
	_ HassAPIObject              = (*HassService)(nil)
	_ zerolog.LogObjectMarshaler = (*HassService)(nil)
)

// HassService is used to call the HASS service API
type HassService struct {
	ID          uint64          `json:"id"`
	Type        string          `json:"type"`
	Domain      string          `json:"domain"`
	Service     string          `json:"service"`
	ServiceData HassServiceData `json:"service_data"`
}

// HassServiceData godoc
type HassServiceData struct {
	EntityID string `json:"entity_id"`
}

// GetID godoc
func (s HassService) GetID() uint64 {
	return s.ID
}

// GetType godoc
func (s HassService) GetType() string {
	return s.Type
}

// Duplicate godoc
func (s HassService) Duplicate(id uint64) HassAPIObject {
	dup := s
	dup.ID = id
	return dup
}

// MarshalZerologObject godoc
func (s HassService) MarshalZerologObject(event *zerolog.Event) {
	event.
		Uint64("id", s.ID).
		Str("type", s.Type).
		Str("domain", s.Domain).
		Str("service", s.Service).
		Str("entity_id", s.ServiceData.EntityID)
}
