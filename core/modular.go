package core

import "time"

// Modular is an interface that will implement a check function
type Modular interface {
	Check()
	GetInterval() time.Duration
	IsEnabled() bool
}
