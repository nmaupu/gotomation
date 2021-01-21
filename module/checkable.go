package module

// Checkable is an interface to check something at a regular interval
type Checkable interface {
	Configure(data interface{}, module Modular) error
	Start() error
	Stop()
}
