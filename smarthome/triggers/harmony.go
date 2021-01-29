package triggers

import (
	"github.com/nmaupu/gotomation/core"
)

var (
	_ core.Actionable = (*Harmony)(nil)
)

// Harmony checks for harmony remote button press and takes action accordingly
type Harmony struct {
	core.Action `mapstructure:",squash"`
}
