package messaging

import (
	"fmt"
	"github.com/nmaupu/gotomation/httpclient"
	"github.com/nmaupu/gotomation/model"
)

var (
	_ Sender = (*StatusLedSender)(nil)
)

type StatusLedSender struct {
	Entity model.HassEntity `mapstructure:"entity" json:"entity"`
}

// Send sends a message via a LED
func (t *StatusLedSender) Send(_ Message, event *model.HassEvent) error {
	if event == nil {
		return fmt.Errorf("event passed to sender is nil")
	}

	action := "turn_off"
	if event.Event.Data.NewState.State == model.StateON {
		action = "turn_on"
	}
	return httpclient.GetSimpleClient().CallService(t.Entity, action, nil)
}
