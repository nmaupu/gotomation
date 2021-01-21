package smarthome

// Triggerable is an interface to trigger an action when a change is detected
type Triggerable interface {
	Configure(data interface{}, action Actionable) error
	GetActionable() Actionable
}
