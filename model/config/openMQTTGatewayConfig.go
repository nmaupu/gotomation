package config

type OpenMQTTGatewayConfig struct {
	MQTT struct {
		Broker   string `mapstructure:"broker"`
		Username string `mapstructure:"username"`
		Password string `mapstructure:"password"`
		Prefix   string `mapstructure:"prefix"`
	} `mapstructure:"mqtt"`
}
