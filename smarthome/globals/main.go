package globals

import (
	"github.com/nmaupu/gotomation/core"
)

var (
	// Coords are home's GPS coordinates
	Coords core.Coordinates
	// Checkers stores all checkers
	Checkers map[string]core.Checkable
	// Triggers stores all triggers
	Triggers map[string]core.Triggerable
)
