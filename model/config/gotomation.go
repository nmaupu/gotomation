package config

// Gotomation is the struct to unmarshal configuration
// It is using mapstructure for a compatibility with Viper config files
type Gotomation struct {
	// LogLevel is the log level configured
	LogLevel string `mapstructure:"log_level"`
	// HomeAssistant server related options
	HomeAssistant struct {
		Host            string   `mapstructure:"host"`
		Token           string   `mapstructure:"token"`
		SubscribeEvents []string `mapstructure:"subscribe_events"`
		HomeZoneName    string   `mapstructure:"home_zone_name"`
	} `mapstructure:"home_assistant"`

	// Modules configuration
	Modules []map[string]interface{} `mapstructure:"modules"`

	// Triggers configuration
	Triggers []map[string]interface{} `mapstructure:"triggers"`

	// Crons configuration
	Crons []interface{} `mapstructure:"crons"`
}
