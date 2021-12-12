package smarthome

import (
	"bytes"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/nmaupu/gotomation/core"
	"github.com/nmaupu/gotomation/logging"
	"github.com/nmaupu/gotomation/model"
	"github.com/nmaupu/gotomation/smarthome/messaging"
	"net/http"
	"strings"
	"text/template"
)

var (
	_ core.Actionable = (*AlertTriggerBool)(nil)
)

const (
	DefaultTemplateString = "{{ .Event.EntityID }} has been changed to {{ .Event.NewState.State }}"
)

// AlertTriggerBool sends alert to a Sender when a specific boolean has its state changed
type AlertTriggerBool struct {
	core.Action `mapstructure:",squash"`
	Sender      string `mapstructure:"sender"`
	// Templates are the template to use to send notification message
	Templates map[string]struct {
		// MsgTemplate is used to format the message sent
		MsgTemplate string `mapstructure:"msg_template"`
	} `mapstructure:"templates"`
}

// Trigger godoc
func (a *AlertTriggerBool) Trigger(event *model.HassEvent) {
	l := logging.NewLogger("AlertTriggerBool.Trigger")

	if event == nil {
		l.Warn().Msg("Event received is nil")
		return
	}

	l = l.With().
		Str("sender_name", a.Sender).
		Str("event_type", event.Event.EventType).
		Str("data.source_name", event.Event.Data.SourceName).
		Str("data.type", event.Event.Data.Type).
		Str("data.key", event.Event.Data.Key).
		Logger()

	l.Debug().Msg("Trigger receiver")

	// Retrieve sender and send message with it
	sender := GetSender(a.Sender)
	if sender == nil {
		l.Error().Msg("sender does not exist")
		return
	}

	entity := event.Event.Data.EntityID

	// Getting template if set
	tplString := DefaultTemplateString
	t, ok := a.Templates[entity]
	if ok && t.MsgTemplate != "" {
		tplString = t.MsgTemplate
	}

	tmpl, err := template.New("messageSender").Parse(tplString)
	if err != nil {
		l.Error().
			Err(err).
			Str("template", tplString).
			Msg("an error occurred compiling template")
		sender.Send(a.getErrorMessage(event, err))
		return
	}
	buf := bytes.NewBufferString("")
	err = tmpl.Execute(buf, struct {
		Event model.HassEventData
	}{
		Event: event.Event.Data,
	})
	if err != nil {
		l.Error().
			Err(err).
			Str("template", tplString).
			Msg("an error occurred executing template")
		sender.Send(a.getErrorMessage(event, err))
		return
	}

	msg := strings.Trim(buf.String(), " ")
	l.Debug().
		Str("msg", msg).
		Msg("Message to send")
	if msg == "" {
		l.Warn().Msg("Message is empty, ignoring event")
		return
	}
	err = sender.Send(messaging.Message{
		Content: msg,
	})
	if err != nil {
		l.Error().
			Err(err).
			Msg("Error sending message to sender")
	}
}

func (a *AlertTriggerBool) getErrorMessage(event *model.HassEvent, err error) messaging.Message {
	return messaging.Message{
		Content: fmt.Sprintf("Error for entity %s, err=%s", event.Event.Data.EntityID, err.Error()),
	}
}

// GinHandler godoc
func (a *AlertTriggerBool) GinHandler(c *gin.Context) {
	c.JSON(http.StatusOK, *a)
}
