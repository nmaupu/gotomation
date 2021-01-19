package module

import (
	"log"
	"time"

	"github.com/go-ping/ping"
	"github.com/mitchellh/mapstructure"
	"github.com/nmaupu/gotomation/httpclient"
	"github.com/nmaupu/gotomation/model"
)

var (
	_ (Checkable) = (*FreeboxChecker)(nil)
)

const (
	defaultRebootEveryMin = 300 * time.Second
)

// FreeboxConfig is the configuration struct for FreeboxChecker module
type FreeboxConfig struct {
	Enabled       bool   `mapstructure:"enabled"`
	PingHost      string `mapstructure:"ping_host"`
	Interval      string `mapstructure:"interval"`
	RestartEntity string `mapstructure:"restart_entity"`
}

// FreeboxChecker module pings a host at a regular interval and restart the freebox if it fails
type FreeboxChecker struct {
	Checker
	// Host is the host to ping
	Host string
	// RebootEveryMin is the min duration between 2 reboots
	RebootEveryMin time.Duration

	entityToRestart model.HassEntity
	lastReboot      time.Time
}

// Configure reads the configuration and returns a new Checkable object
func (c *FreeboxChecker) Configure(data interface{}) error {
	config := FreeboxConfig{}
	err := mapstructure.Decode(data, &config)
	if err != nil {
		return err
	}

	c.Host = config.PingHost
	c.RebootEveryMin = defaultRebootEveryMin
	c.entityToRestart = model.NewHassEntity(config.RestartEntity)

	interval, err := time.ParseDuration(config.Interval)
	if err != nil {
		return err
	}

	c.Checker = Checker{
		Enabled:  config.Enabled,
		Interval: interval,
		Checker:  c,
	}

	return nil
}

// Check runs a single check
func (c *FreeboxChecker) Check() {
	if c.RebootEveryMin == 0 {
		c.RebootEveryMin = defaultRebootEveryMin
	}
	log.Printf("[FreeboxChecker] Checking for freebox health (ping %s)", c.Host)
	pinger, _ := ping.NewPinger(c.Host)

	pinger.Count = 1
	pinger.Timeout = 2 * time.Second // timeout for all pings to be performed !
	pinger.SetPrivileged(false)

	err := pinger.Run()
	if err != nil {
		log.Println("An error occurred creating pinger object")
		return
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
		httpclient.SimpleClientSingleton.CallService(c.entityToRestart, "turn_off")
		time.Sleep(1 * time.Second)
		httpclient.SimpleClientSingleton.CallService(c.entityToRestart, "turn_on")
		c.lastReboot = time.Now()
	} else if !isTimeBetweenRebootOK {
		log.Printf("[FreeboxChecker] Fail but too soon to reboot... %+v", stats)
	}
}
