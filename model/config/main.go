package config

// Gotomation is the struct to unmarshal configuration
// It is using mapstructure for a compatibility with Viper config files
type Gotomation struct {
	// HomeAssistant server related options
	HomeAssistant struct {
		Host  string `mapstructure:"host"`
		Token string `mapstructure:"token"`
	} `mapstructure:"home_assistant"`

	// Modules configuration
	Modules []map[string]interface{} `mapstructure:"modules"`

	// Triggers configuration
	Triggers []map[string]interface{} `mapstructure:"triggers"`
}
