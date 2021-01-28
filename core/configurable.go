package core

// Configurable is an interface to make any object configurable using a config file
type Configurable interface {
	Configure(config interface{}, obj interface{}) error
}
