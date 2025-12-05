package smarthome

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gin-gonic/gin"
	"github.com/nmaupu/gotomation/core"
	"github.com/nmaupu/gotomation/logging"
)

var (
	_ (core.Modular) = (*OpenMQTTGatewayWBListChecker)(nil)
)

type gatewayMacs struct {
	Gateway string   `mapstructure:"gateway"`
	Macs    []string `mapstructure:"macs"`
}

// OpenMQTTGatewayWBListChecker module registers white/black list on a regular interval
type OpenMQTTGatewayWBListChecker struct {
	core.Module `mapstructure:",squash"`
	BlackList   []gatewayMacs `mapstructure:"blacklist"`
	WhiteList   []gatewayMacs `mapstructure:"whitelist"`
}

// Check runs a single check
func (c *OpenMQTTGatewayWBListChecker) Check() {
	l := logging.NewLogger("OpenMQTTGatewayWBListChecker.Check")

	omgConfig := GetOMGConfig()

	// Handling connection to the broker
	opts := mqtt.NewClientOptions()
	opts.AddBroker(omgConfig.MQTT.Broker).
		SetClientID("gotomation_mqtt_client").
		SetUsername(omgConfig.MQTT.Username).
		SetPassword(omgConfig.MQTT.Password)
	client := mqtt.NewClient(opts)
	defer client.Disconnect(0)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		l.Error().Err(token.Error()).Msg("Unable to connect to the MQTT broker")
		return
	}

	// local types
	type payload struct {
		Whitelist []string `json:"white-list"`
		Blacklist []string `json:"black-list"`
	}
	newPayload := func() *payload {
		return &payload{
			// Ensuring lists are empty (not nil)
			Whitelist: []string{},
			Blacklist: []string{},
		}
	}

	gwPayloads := map[string]*payload{}

	// publish lists to relevant topics
	for _, gm := range c.BlackList {
		p, ok := gwPayloads[gm.Gateway]
		if !ok {
			p = newPayload()
		}
		p.Blacklist = gm.Macs
		gwPayloads[gm.Gateway] = p
	}
	for _, gm := range c.WhiteList {
		p, ok := gwPayloads[gm.Gateway]
		if !ok {
			p = newPayload()
		}
		p.Whitelist = gm.Macs
		gwPayloads[gm.Gateway] = p
	}

	for gw, v := range gwPayloads {
		payloadJSON, _ := json.Marshal(v)
		c.publishConfig(client, omgConfig.MQTT.Prefix, gw, string(payloadJSON))
	}
}

func (c *OpenMQTTGatewayWBListChecker) publishConfig(client mqtt.Client, prefix string, gw string, payload string) {
	l := logging.NewLogger("OpenMQTTGatewayWBListChecker.publishConfig").With().
		Str("gateway", gw).
		Str("payload", payload).Logger()

	// -> home/gw/commands/MQTTtoBT/config
	tok := client.Publish(
		fmt.Sprintf("%s/%s/commands/MQTTtoBT/config", prefix, gw), 0, true, payload)

	if tok.WaitTimeout(time.Second * 5) {
		l.Info().Msg("Config has been published")
	} else {
		l.Error().Msg("Config cannot be published - timeout occurred")
	}
}

// GinHandler godoc
func (c *OpenMQTTGatewayWBListChecker) GinHandler(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, *c)
}
