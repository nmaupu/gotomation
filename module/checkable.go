package module

// Checkable is an interface to check something at a regular interval
type Checkable interface {
	Configure(interface{}) error
	Check()
	Start()
	Stop()
	IsEnabled() bool
}
