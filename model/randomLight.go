package model

import "time"

// RandomLight is used to randomly set to on/off a light
type RandomLight struct {
	Entity      HassEntity    `mapstructure:"entity"`
	MinDuration time.Duration `mapstructure:"min_duration"`
	MaxDuration time.Duration `mapstructure:"max_duration"`
}
