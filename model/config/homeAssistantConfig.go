package config

import "github.com/nmaupu/gotomation/model"

type HomeAssistantConfig struct {
	Host                string             `mapstructure:"host"`
	Token               string             `mapstructure:"token"`
	SubscribeEvents     []string           `mapstructure:"subscribe_events"`
	HomeZoneName        string             `mapstructure:"home_zone_name"`
	TLSEnabled          bool               `mapstructure:"tls_enabled"`
	HealthCheckEntities []model.HassEntity `mapstructure:"health_check_entities"`
}
