package module

// Checkable is an interface to check something at a regular interval
type Checkable interface {
	Configure(data interface{}, destImpl Checkable) error
	Check()
	Start(checkFunc func())
	Stop()
	IsEnabled() bool
}
