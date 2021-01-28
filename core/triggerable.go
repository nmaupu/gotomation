package core

// Triggerable is an interface to trigger an action when a change is detected
type Triggerable interface {
	Configurable
	GetActionable() Actionable
}
