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
	"reflect"
	"strconv"
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
		// BoolAttributeKey is the state attribute name used to get real entity state, if not set, using event's state directly
		// An event is ignored when old state == new state
		BoolAttributeKey string `mapstructure:"bool_attr_key"`
		// BoolAttributeValueTrue is the value to compare with to get the actual state
		BoolAttributeValueTrue string `mapstructure:"bool_attr_val_true"`
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

	/*newState := a.getState(event, true)
	oldState := a.getState(event, false)
	if newState == oldState {
		l.Warn().Msg("Old state and new state are identical, ignoring event")
		return
	}*/

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

// getState returns the new or old state of the entity
// When using an input_boolean, state is given by the State field directly
// When using another type of hardware sensor, state field can be irrelevant and state can be encoded into attributes
// e.g: Aqara water sensor encodes it in attribute "water_leak" as a boolean
func (a *AlertTriggerBool) getState(event *model.HassEvent, new bool) bool {
	entity := event.Event.Data.EntityID
	t, ok := a.Templates[entity]
	if !ok || t.BoolAttributeKey == "" {
		return event.Event.Data.NewState.IsON()
	}

	valTrue := "true"
	if t.BoolAttributeValueTrue != "" {
		valTrue = t.BoolAttributeValueTrue
	}

	// Looking for a specific attribute
	var attr interface{}
	if new {
		attr, ok = event.Event.Data.NewState.Attributes[t.BoolAttributeKey]
	} else {
		attr, ok = event.Event.Data.OldState.Attributes[t.BoolAttributeKey]
	}
	if !ok {
		return false
	}

	if reflect.TypeOf(attr).Kind() == reflect.String {
		return attr.(string) == valTrue
	}
	if reflect.TypeOf(attr).Kind() == reflect.Bool {
		vt, err := strconv.ParseBool(valTrue)
		if err != nil {
			vt = false
		}
		return attr.(bool) == vt
	}

	return false
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
