package model

// HassAPIObject is the generic interface for different message type
type HassAPIObject interface {
	GetID() uint64
	GetType() string
	Duplicate(uint64) HassAPIObject
}
