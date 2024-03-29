package smarthome

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"time"

	"github.com/nmaupu/gotomation/core"
	"github.com/nmaupu/gotomation/logging"
	"github.com/nmaupu/gotomation/thirdparty"
	"google.golang.org/api/calendar/v3"
)

var (
	_ (core.Modular) = (*CalendarChecker)(nil)
)

// CalendarChecker checks calendar for new events once in a while
type CalendarChecker struct {
	core.Module `mapstructure:",squash"`
	Cals        []struct {
		Name string `mapstructure:"name"`
		ID   string `mapstructure:"id"`
	} `mapstructure:"cals"`
}

// Check runs a single check
func (c *CalendarChecker) Check() {
	l := logging.NewLogger("CalendarLights.Check")

	client, err := thirdparty.GetGoogleConfig().GetClient()
	if err != nil {
		l.Error().Err(err).Msg("Unable to get google's API client")
		return
	}
	srv, err := calendar.New(client)
	if err != nil {
		l.Error().Err(err).Msg("Unable to get google's API client")
		return
	}

	now := time.Now().Local().Format(time.RFC3339)
	for _, cal := range c.Cals {
		events, err := srv.Events.List(cal.ID).
			ShowDeleted(false).
			SingleEvents(true).
			TimeMin(now).
			MaxResults(10).
			OrderBy("startTime").Do()
		if err != nil {
			l.Error().Err(err).Str("cal_name", cal.Name).Msg("Unable to get events from calendar")
			continue
		}

		for _, item := range events.Items {
			dateStart := item.Start.DateTime
			if dateStart == "" {
				dateStart = item.Start.Date
			}
			dateEnd := item.End.DateTime
			if dateEnd == "" {
				dateEnd = item.End.Date
			}

			l.Debug().Str("cal_name", cal.Name).
				Str("summary", item.Summary).
				Str("content", item.Description).
				Str("date_beg", dateStart).
				Str("date_end", dateEnd).
				Msg("Calendar event")
		}
	}
}

func (c *CalendarChecker) GinHandler(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, *c)
}
