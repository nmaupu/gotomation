package config

import (
	"fmt"

	"github.com/nmaupu/gotomation/smarthome/messaging"
	"github.com/rs/zerolog"
)

type SenderConfig struct {
	// Name of this config
	Name string `mapstructure:"name" json:"name"`
	// Telegram configures a telegram sender
	Telegram *messaging.TelegramSender `mapstructure:"telegram" json:"telegram"`
	// StatusLed configures a status LED
	StatusLed *messaging.StatusLedSender `mapstructure:"statusLed" json:"statusLed"`
}

// GetSender gets the Sender interface depending on what field is set
func (s *SenderConfig) GetSender() (messaging.Sender, error) {
	if s.Telegram != nil {
		if s.Telegram.Token == "" || s.Telegram.ChatID == 0 {
			return nil, fmt.Errorf("error creating Telegram config, token or char_id is unspecified for %s", s.Name)
		}

		return s.Telegram, nil
	}

	if s.StatusLed != nil {
		return s.StatusLed, nil
	}

	return nil, fmt.Errorf("no sender specified in configuration for %s", s.Name)
}

func (s SenderConfig) MarshalZerologObject(event *zerolog.Event) {
	event.
		Str("name", s.Name)
	if s.Telegram != nil {
		event.
			Int64("telegram_chat_id", s.Telegram.ChatID).
			Str("telegram_token", s.Telegram.Token)
	}
	if s.StatusLed != nil {
		event.Object("status_led_", s.StatusLed.Entity)
	}
}
