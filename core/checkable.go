package core

import "github.com/nmaupu/gotomation/routines"

// Checkable is an interface to check something at a regular interval
type Checkable interface {
	Configurable
	routines.Runnable
}
