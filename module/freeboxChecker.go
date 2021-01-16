package module

import (
	"log"
	"time"

	"github.com/go-ping/ping"
	"github.com/nmaupu/gotomation/app"
	"github.com/nmaupu/gotomation/model"
)

const (
	defaultRebootEveryMin = 300 * time.Second
)

// FreeboxChecker module pings a host at a regular interval and restart the freebox if it fails
type FreeboxChecker struct {
	// Every checks for connection every Every
	Every time.Duration
	// Host is the host to ping
	Host string
	// RebootEveryMin is the min duration between 2 reboots
	RebootEveryMin time.Duration

	entityToRestart model.HassEntity
	lastReboot      time.Time
}

// NewFreeboxChecker creates a pointer to a FreeboxChecker object
func NewFreeboxChecker(every time.Duration, host string, entity model.HassEntity) *FreeboxChecker {
	return &FreeboxChecker{
		Every:           every,
		Host:            host,
		entityToRestart: entity,
		RebootEveryMin:  defaultRebootEveryMin,
	}
}

// Start starts the module
func (c *FreeboxChecker) Start() error {
	if c.RebootEveryMin == 0 {
		c.RebootEveryMin = defaultRebootEveryMin
	}

	go func() {
		for {
			time.Sleep(c.Every)

			log.Printf("[FreeboxChecker] Checking for freebox health (ping %s)", c.Host)
			pinger, _ := ping.NewPinger(c.Host)

			pinger.Count = 1
			pinger.Timeout = 2 * time.Second // timeout for all pings to be performed !
			pinger.SetPrivileged(false)

			err := pinger.Run()
			if err != nil {
				log.Println("An error occurred creating pinger object")
				continue
			}

			stats := pinger.Statistics()
			isTimeBetweenRebootOK := time.Now().Sub(c.lastReboot) > c.RebootEveryMin
			if stats.PacketLoss == 0 {
				log.Println("[FreeboxChecker] Freebox OK!")
			} else if stats.PacketLoss > 0 && stats.PacketLoss != 100 {
				log.Printf("[FreeboxChecker] Some packet are lost, statistics=%+v", stats)
			} else if stats.PacketLoss == 100 && isTimeBetweenRebootOK {
				log.Println("[FreeboxChecker] 100% packet lost, rebooting fbx...")
				// Rebooting
				app.GetSimpleClient().CallService(c.entityToRestart, "turn_off")
				time.Sleep(1 * time.Second)
				app.GetSimpleClient().CallService(c.entityToRestart, "turn_on")
				c.lastReboot = time.Now()
			} else if !isTimeBetweenRebootOK {
				log.Printf("[FreeboxChecker] Fail but too soon to reboot... %+v", stats)
			}
		}
	}()

	return nil
}
