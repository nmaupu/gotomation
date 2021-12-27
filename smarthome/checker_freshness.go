package smarthome

import (
	"bytes"
	"fmt"
	"github.com/nmaupu/gotomation/httpclient"
	"github.com/nmaupu/gotomation/model"
	"github.com/nmaupu/gotomation/smarthome/messaging"
	"strings"
	"text/template"
	"time"

	"github.com/nmaupu/gotomation/core"
	"github.com/nmaupu/gotomation/logging"
)

var (
	_ core.Modular = (*FreshnessChecker)(nil)
)

const (
	DefaultTimeFormat                     = time.RFC3339
	DefaultFreshnessCheckerTemplateString = `The following entities have not been seen since {{ .Checker.Freshness }}: {{ JoinEntities .Entities ", " }}`
)

// FreshnessChecker checks freshness of devices against a duration (using last_seen property)
type FreshnessChecker struct {
	core.Module `mapstructure:",squash"`
	// Name for this FreshnessChecker
	Name string `mapstructure:"name"`
	// Entities are the entities returning device's last seen value
	Entities []model.HassEntity `mapstructure:"entities"`
	// Sender is used to send an alert if one or more entities have not been seen soon enough
	Sender string `mapstructure:"sender"`
	// Freshness configures the max allowed time a device has to be seen
	Freshness time.Duration `mapstructure:"freshness"`
	// TimeFormat sets the time format if different from default
	TimeFormat string `mapstructure:"time_format"`
	// Template to use for sending message to the sender
	Template string `mapstructure:"template"`
}

// Check runs a single check
func (c *FreshnessChecker) Check() {
	l := logging.NewLogger("FreshnessChecker.Check")

	l.Debug().Msg("Checking all devices")

	if c.TimeFormat == "" {
		c.TimeFormat = DefaultTimeFormat
	}

	now := time.Now()

	notFreshEntities := make([]model.HassEntity, 0)

	for _, entity := range c.Entities {
		hassEntity, err := httpclient.GetSimpleClient().GetEntity(entity.Domain, entity.EntityID)
		if err != nil {
			l.Error().
				Err(err).
				Object("entity", entity).
				Msg("unable to get entity")
			continue
		}

		lastSeen, err := time.Parse(c.TimeFormat, hassEntity.State.State)
		if err != nil {
			l.Error().Err(err).
				Object("entity", entity).
				Str("last_seen", hassEntity.State.State).
				Str("time_format", c.TimeFormat).
				Msg("unable to parse last seen time")
			continue
		}

		duration := now.Sub(lastSeen)
		if duration > c.Freshness {
			l.Warn().
				Object("entity", entity).
				Str("freshness", c.Freshness.String()).
				Time("last_seen", lastSeen).
				Str("sender", c.Sender).
				Msg("entity has not been seen recently !")
			notFreshEntities = append(notFreshEntities, hassEntity)
		}

		// ok
		l.Debug().
			Object("entity", entity).
			Str("last_seen_duration", duration.String()).
			Msg("Duration since last seen date")
	}

	// Creating a message for the sender if something is to be sent
	if len(notFreshEntities) == 0 {
		l.Debug().
			Str("name", c.Name).
			Msg("All entities are ok !")
		return
	}

	// Prepare warning message to send
	sender := GetSender(c.Sender)
	if c.Template == "" {
		c.Template = DefaultFreshnessCheckerTemplateString
	}
	tmpl, err := template.New("messageSender").
		Funcs(template.FuncMap{"JoinEntities": model.JoinEntities}).
		Parse(c.Template)
	if err != nil {
		l.Error().
			Err(err).
			Str("template", c.Template).
			Msg("an error occurred compiling template")
		if err := sender.Send(c.getErrorMessage(err)); err != nil {
			l.Error().Err(err).
				Str("sender", c.Sender).
				Msg("unable to send message to sender")
		}
		return
	}

	buf := bytes.NewBufferString("")
	err = tmpl.Execute(buf, struct {
		Checker  FreshnessChecker
		Entities []model.HassEntity
	}{
		Checker:  *c,
		Entities: notFreshEntities,
	})
	if err != nil {
		l.Error().
			Err(err).
			Str("template", c.Template).
			Msg("an error occurred executing template")
		if err := sender.Send(c.getErrorMessage(err)); err != nil {
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
	})
	if err != nil {
		l.Error().
			Err(err).
			Msg("Error sending message to sender")
	}

}

func (c *FreshnessChecker) getErrorMessage(err error) messaging.Message {
	return messaging.Message{
		Content: fmt.Sprintf("FreshnessChecker: cannot compile template %s, err=%s", c.Template, err.Error()),
	}
}
