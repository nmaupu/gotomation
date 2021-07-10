package core

import (
	"fmt"
	"github.com/nmaupu/gotomation/app"
	"github.com/nmaupu/gotomation/httpclient"
	"github.com/nmaupu/gotomation/logging"
	"github.com/nmaupu/gotomation/model"
	"github.com/nmaupu/gotomation/routines"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

type RandomLightsRoutine interface {
	routines.Runnable
	GetName() string
	GetSlots() uint32
}

func NewRandomLightsRoutine(name string, slots uint32, lights []model.RandomLight, startTime, endTime time.Time, odds uint32, refreshEvery time.Duration) (RandomLightsRoutine, error) {
	if name == "" {
		return nil, fmt.Errorf("name must not be empty")
	}

	if slots < 1 {
		return nil, fmt.Errorf("slotsCh must be >= 1")
	}

	return &randomLightsRoutine{
		name:         name,
		nbSlots:      slots,
		lights:       lights,
		startTime:    startTime,
		endTime:      endTime,
		odds:         odds,
		refreshEvery: refreshEvery,
	}, nil
}

type autoLight struct {
	model.RandomLight
	duration time.Duration
}

type randomLightsRoutine struct {
	name                 string
	started              bool
	mutexStopStart       sync.Mutex
	mainRoutineDone      chan bool
	autoLightRoutineDone chan bool
	nbSlots              uint32

	// lights are all the lights available
	lights []model.RandomLight
	// startTime is the time when starting to automatically set lights to on/off
	startTime time.Time
	// endTime is the time when stopping to automatically set lights to on/off
	endTime time.Time
	// odds configures the chances a light has to be turned on
	odds uint32
	// refreshEvery configures the duration between each refresh call
	refreshEvery time.Duration

	// autoLightsCh stores messages for lights to be set to on if slotsCh are available
	autoLightsCh chan autoLight
	// slotsCnt counts how many slots have been reserved
	slotsCnt uint32
}

func (r *randomLightsRoutine) IsAutoStart() bool {
	return false
}

func (r *randomLightsRoutine) Start() error {
	r.mutexStopStart.Lock()
	defer r.mutexStopStart.Unlock()
	if r.started {
		return nil
	}

	l := logging.NewLogger("randomLightsRoutine.Start")

	r.mainRoutineDone = make(chan bool, 1)
	r.autoLightRoutineDone = make(chan bool, 1)
	r.autoLightsCh = make(chan autoLight) // main chan to process messages

	// Starting autoLightRoutine which listens for incoming messages
	app.RoutinesWG.Add(1)
	go func() {
		defer app.RoutinesWG.Done()
		r.autoLightRoutine()
	}()

	// first init before ticker ticks
	app.RoutinesWG.Add(1)
	go func() {
		defer app.RoutinesWG.Done()
		r.refresh()
	}()

	app.RoutinesWG.Add(1)
	go func() {
		defer app.RoutinesWG.Done()
		l.Debug().Msg("Starting randomLightsRoutine refresh go routine")
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-r.mainRoutineDone:
				l.Trace().Msg("Exiting randomLightsRoutine refresh go routine")
				return
			case <-ticker.C:
				r.refresh()
			}
		}
	}()

	r.started = true
	return nil
}

func (r *randomLightsRoutine) refresh() {
	l := logging.NewLogger("randomLightsRoutine.refresh")

	// time check (begin < now < end)
	// Generate light
	// Generate duration depending on the light
	// send message

	sunrise, _, err := Coords().GetSunriseSunset()
	if err != nil {
		l.Error().Err(err).Msg("unable to get sunrise time")
		return
	}

	now := time.Now()
	if !r.CheckTime(now, sunrise) {
		l.Trace().
			Time("start", r.startTime).
			Time("end", r.endTime).
			Time("sunrise", sunrise).
			Time("now", now).
			Msg("Current time is outside of time range, nothing to do")
		return
	}

	val := rand.Int()
	oddLog := l.With().
		Uint32("odds", r.odds).
		Int("rand", val).
		Int("val%Odds", val%int(r.odds)).
		Logger()
	if val%int(r.odds) != 0 {
		oddLog.Debug().Msg("Not setting a light ON")
		return
	}
	oddLog.Debug().Msg("Setting a light ON")

	light := r.lights[rand.Intn(len(r.lights))]
	minDurSec := int64(light.MinDuration / time.Second)
	maxDurSec := int64(light.MaxDuration / time.Second)
	msg := autoLight{
		RandomLight: light,
		duration:    time.Duration(rand.Int63n(maxDurSec-minDurSec)+minDurSec) * time.Second,
	}
	r.autoLightsCh <- msg
}

func (r *randomLightsRoutine) autoLightRoutine() {
	l := logging.NewLogger("randomLightsRoutine.autoLightRoutine")

	// Init lights status map
	lightsStatus := make(map[string]bool)
	for _, light := range r.lights {
		lightsStatus[light.Entity.GetEntityIDFullName()] = false
	}

	// Routine loop
	for {
		select {
		case msg := <-r.autoLightsCh:
			// Verify status of current light
			if lightsStatus[msg.Entity.GetEntityIDFullName()] { // light is already on
				l.Debug().
					EmbedObject(msg.Entity).
					Str("duration", msg.duration.String()).
					Msg("Light is already ON, ignoring")
				continue
			}

			// Reserving a slot
			if !r.reserveSlot() { // ignore message
				l.Debug().
					EmbedObject(msg.Entity).
					Str("duration", msg.duration.String()).
					Msg("No slot available, ignoring")
				continue
			}

			l.Debug().
				EmbedObject(msg.Entity).
				Str("duration", msg.duration.String()).
				Msg("Slot available, setting a light ON")

			l.Debug().EmbedObject(msg.Entity).Msg("Setting light to ON")
			err := httpclient.GetSimpleClient().CallService(msg.Entity, "turn_on", map[string]interface{}{
				"brightness": 90,
			})
			if err != nil {
				l.Error().Err(err).EmbedObject(msg.Entity).Msg("unable to turn_on light")
				continue
			}
			lightsStatus[msg.Entity.GetEntityIDFullName()] = true

			time.AfterFunc(msg.duration, func() {
				err := httpclient.GetSimpleClient().CallService(msg.Entity, "turn_off", nil)
				if err != nil {
					l.Error().Err(err).EmbedObject(msg.Entity).Msg("unable to turn_off light")
					// here we ignore the error, log only to get a trace of the issue
				}
				lightsStatus[msg.Entity.GetEntityIDFullName()] = false
				r.freeSlot()
			})

		case <-r.autoLightRoutineDone:
			l.Trace().Msg("Exiting randomLightsRoutine.autoLightRoutine go routine")
			return
		}
	}
}

// reserveSlot tries to push a value to a channel, returns true if it succeeds, false otherwise
func (r *randomLightsRoutine) reserveSlot() bool {
	if atomic.LoadUint32(&r.slotsCnt) >= r.nbSlots {
		return false
	}

	atomic.AddUint32(&r.slotsCnt, 1)
	return true
}

func (r *randomLightsRoutine) freeSlot() {
	val := atomic.LoadUint32(&r.slotsCnt)
	if val > 0 {
		atomic.StoreUint32(&r.slotsCnt, val-1)
	}
}

func (r *randomLightsRoutine) Stop() {
	r.mutexStopStart.Lock()
	defer r.mutexStopStart.Unlock()
	if !r.started {
		return
	}
	r.mainRoutineDone <- true
	r.autoLightRoutineDone <- true
	r.started = false
}

func (r *randomLightsRoutine) GetName() string {
	return fmt.Sprintf("randomLightsRoutine.%s", r.name)
}

func (r *randomLightsRoutine) GetSlots() uint32 {
	return r.nbSlots
}

func (r *randomLightsRoutine) IsStarted() bool {
	r.mutexStopStart.Lock()
	defer r.mutexStopStart.Unlock()
	return r.started
}

// CheckTime checks if current time is ok to switch lights on
func (r *randomLightsRoutine) CheckTime(now time.Time, sunrise time.Time) bool {
	start := time.Date(now.Year(), now.Month(), now.Day(), r.startTime.Hour(), r.startTime.Minute(), r.startTime.Second(), 0, now.Location())
	if now.Before(sunrise) {
		start = start.Add(-24 * time.Hour)
	}

	end := time.Date(now.Year(), now.Month(), now.Day(), r.endTime.Hour(), r.endTime.Minute(), r.endTime.Second(), 0, now.Location())
	if end.Before(start) { // end is after midnight
		end = end.Add(24 * time.Hour)
	}

	return start.Before(now) && end.After(now)
}
