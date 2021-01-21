package module

import (
	"log"
	"time"

	"github.com/go-ping/ping"
	"github.com/nmaupu/gotomation/httpclient"
	"github.com/nmaupu/gotomation/model"
)

var (
	_ (Modular) = (*InternetChecker)(nil)
)

const (
	defaultRebootEveryMin = 300 * time.Second
)

// InternetChecker module pings a host at a regular interval and restart the internet box if it fails
type InternetChecker struct {
	Module `mapstructure:",squash"`
	// PingHost is the host to ping
	PingHost string `mapstructure:"ping_host"`
	// MaxRebootEvery is the min duration between 2 reboots
	MaxRebootEvery time.Duration `mapstructure:"max_reboot_every"`
	// RestartEntity is the entity to restart in case of failure
	RestartEntity model.HassEntity `mapstructure:"restart_entity"`
	// lastReboot stores the last reboot time
	lastReboot time.Time
}

// Check runs a single check
func (c *InternetChecker) Check() {
	if c.MaxRebootEvery == 0 {
		c.MaxRebootEvery = defaultRebootEveryMin
	}
	log.Printf("[InternetChecker] Checking for internet health (ping %s)", c.PingHost)
	pinger, _ := ping.NewPinger(c.PingHost)

	pinger.Count = 1
	pinger.Timeout = 2 * time.Second // timeout for all pings to be performed !
	pinger.SetPrivileged(false)

	err := pinger.Run()
	if err != nil {
		log.Println("An error occurred creating pinger object")
		return
	}

	stats := pinger.Statistics()
	isTimeBetweenRebootOK := time.Now().Sub(c.lastReboot) > c.MaxRebootEvery
	if stats.PacketLoss == 0 {
		log.Println("[InternetChecker] Freebox OK!")
	} else if stats.PacketLoss > 0 && stats.PacketLoss != 100 {
		log.Printf("[InternetChecker] Some packet are lost, statistics=%+v", stats)
	} else if stats.PacketLoss == 100 && isTimeBetweenRebootOK {
		log.Println("[InternetChecker] 100% packet lost, rebooting fbx...")
		// Rebooting
		httpclient.SimpleClientSingleton.CallService(c.RestartEntity, "turn_off")
		time.Sleep(1 * time.Second)
		httpclient.SimpleClientSingleton.CallService(c.RestartEntity, "turn_on")
		c.lastReboot = time.Now()
	} else if !isTimeBetweenRebootOK {
		log.Printf("[InternetChecker] Fail but too soon to reboot... %+v", stats)
	}
}
