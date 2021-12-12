package messaging

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var (
	_ Sender = (*TelegramSender)(nil)
)

type TelegramSender struct {
	Token  string `mapstructure:"token"`
	ChatID int64  `mapstructure:"chat_id"`
}

// Send sends a message to Telegram
func (t *TelegramSender) Send(m Message) error {
	botAPI, err := tgbotapi.NewBotAPI(t.Token)
	if err != nil {
		return err
	}

	_, err = botAPI.Send(tgbotapi.NewMessage(t.ChatID, m.Content))
	return err
}
