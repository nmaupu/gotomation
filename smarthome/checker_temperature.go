package smarthome

import (
	"bytes"
	"fmt"
	"github.com/nmaupu/gotomation/core"
	"github.com/nmaupu/gotomation/httpclient"
	"github.com/nmaupu/gotomation/logging"
	"github.com/nmaupu/gotomation/model"
	"github.com/nmaupu/gotomation/smarthome/messaging"
	"strconv"
	"strings"
	"text/template"
	"time"
)

var (
	_ core.Modular = (*TemperatureChecker)(nil)
)

const (
	DefaultTemperatureCheckerTempThreshold       = 25
	DefaultTemperatureCheckerSendMessageInterval = 24 * time.Hour
	DefaultTemperatureCheckerTemplate            = `The following sensors have exceeded the temperature threshold: {{ JoinEntities .Entities ", " }}`
)

type TemperatureChecker struct {
	core.Module `mapstructure:",squash"`
	// Sensors are the entities to check for temperature threshold
	Sensors []struct {
		Entity        model.HassEntity `mapstructure:"entity"`
		TempThreshold float64          `mapstructure:"temp_threshold"`
	} `mapstructure:"sensors"`
	DateBegin           model.DayMonthDate `mapstructure:"date_begin"`
	DateEnd             model.DayMonthDate `mapstructure:"date_end"`
	Sender              string             `mapstructure:"sender"`
	SendMessageInterval time.Duration      `mapstructure:"send_message_interval"`
	Template            string             `mapstructure:"template"`

	lastMessageSentTime map[string]time.Time
}

func (c *TemperatureChecker) Check() {
	l := logging.NewLogger("TemperatureChecker.Check")

	l.Debug().Msg("Checking all sensors")

	now := time.Now()
	if c.DateBegin.After(now) && c.DateEnd.Before(now) {
		l.Debug().
			Time("current", now).
			Time("begin_date", time.Time(c.DateBegin)).
			Time("end_date", time.Time(c.DateEnd)).
			Msg("Current date is NOT between begin and end, nothing to do")
		return
	}

	if c.lastMessageSentTime == nil {
		c.lastMessageSentTime = make(map[string]time.Time)
	}

	problematicEntities := make([]model.HassEntity, 0)

	for _, sensor := range c.Sensors {
		entity := sensor.Entity
		tempThreshold := sensor.TempThreshold
		if tempThreshold <= 0 {
			tempThreshold = DefaultTemperatureCheckerTempThreshold
		}
		sendMsgInterval := c.SendMessageInterval
		if sendMsgInterval <= 0 {
			sendMsgInterval = DefaultTemperatureCheckerSendMessageInterval
		}

		hassEntity, err := httpclient.GetSimpleClient().GetEntity(entity.Domain, entity.EntityID)
		if err != nil {
			l.Error().
				Err(err).
				Object("entity", entity).
				Msg("unable to get entity")
			continue
		}

		temp, err := strconv.ParseFloat(hassEntity.State.State, 64)
		if err != nil {
			l.Error().
				Err(err).
				Object("entity", entity).
				Str("state", hassEntity.State.State).
				Msg("unable to convert state to float")
			continue
		}

		lastSent, ok := c.lastMessageSentTime[hassEntity.GetEntityIDFullName()]
		if ok && lastSent.Add(c.SendMessageInterval).After(now) {
			// ignoring event, too close to the previous one
			continue
		}

		if temp > sensor.TempThreshold {
			problematicEntities = append(problematicEntities, hassEntity)
		}
	}

	// Send message
	if len(problematicEntities) == 0 {
		// Nothing to do
		return
	}

	// Update date for all message sent for those "problematic" entities
	for _, e := range problematicEntities {
		c.lastMessageSentTime[e.GetEntityIDFullName()] = now
	}

	sender := GetSender(c.Sender)
	if c.Template == "" {
		c.Template = DefaultTemperatureCheckerTemplate
	}
	tmpl, err := template.New("messageSender").
		Funcs(template.FuncMap{"JoinEntities": model.JoinEntities}).
		Parse(c.Template)
	if err != nil {
		l.Error().
			Err(err).
			Str("template", c.Template).
			Msg("an error occurred compiling template")
		if err := sender.Send(c.getErrorMessage(err), nil); err != nil {
			l.Error().Err(err).
				Str("sender", c.Sender).
				Msg("unable to send message to sender")
		}
		return
	}

	buf := bytes.NewBufferString("")
	err = tmpl.Execute(buf, struct {
		Checker  TemperatureChecker
		Entities []model.HassEntity
	}{
		Checker:  *c,
		Entities: problematicEntities,
	})
	if err != nil {
		l.Error().
			Err(err).
			Str("template", c.Template).
			Msg("an error occurred executing template")
		if err := sender.Send(c.getErrorMessage(err), nil); err != nil {
			l.Error().Err(err).
				Str("sender", c.Sender).
				Msg("unable to send message to sender")
		}
		return
	}

	msg := strings.Trim(buf.String(), " ")
	l.Debug().
		Str("msg", msg).
		Str("sender", c.Sender).
		Msg("Message to send")
	if msg == "" {
		l.Warn().Msg("Message is empty, ignoring event")
		return
	}
	err = sender.Send(messaging.Message{
		Content: msg,
	}, nil)
	if err != nil {
		l.Error().
			Err(err).
			Msg("Error sending message to sender")
	}

}

func (c *TemperatureChecker) getErrorMessage(err error) messaging.Message {
	return messaging.Message{
		Content: fmt.Sprintf("TemperatureChecker: cannot compile template %s, err=%s", c.Template, err.Error()),
	}
}
