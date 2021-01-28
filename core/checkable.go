package core

// Checkable is an interface to check something at a regular interval
type Checkable interface {
	Configurable
	Start() error
	Stop()
}
