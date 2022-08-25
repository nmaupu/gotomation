package messaging

import "github.com/nmaupu/gotomation/model"

type Sender interface {
	Send(m Message, event *model.HassEvent) error
}
