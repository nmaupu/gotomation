package config

// Gotomation is the struct to unmarshal configuration
// It is using mapstructure for a compatibility with Viper config files
type Gotomation struct {
	HomeAssistant struct {
		Host  string `mapstructure:"host"`
		Token string `mapstructure:"token"`
	} `mapstructure:"home_assistant"`

	Modules []map[string]interface{} `mapstructure:"modules"`
}
