package main

import (
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/nmaupu/gotomation/app"
	"github.com/nmaupu/gotomation/model"
	"github.com/nmaupu/gotomation/model/config"
	"github.com/nmaupu/gotomation/module"
	"github.com/spf13/viper"
)

type messageReceived struct {
	Jsonrpc string
	Method  string
}

func main() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/gotomation")
	viper.AddConfigPath("$HOME/.gotomation")
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Fatalf("No config file available")
		}

		log.Fatalf("Cannot read config file, err=%v", err)
	}

	gotomationConfig := config.Gotomation{}
	if err := viper.Unmarshal(&gotomationConfig); err != nil {
		log.Fatalf("Unable to unmarshal config file, err=%v", err)
	}

	// Init daemon service
	app.Init(gotomationConfig)

	// Adding callbacks for server communication
	app.GetWebSocketClient().RegisterCallback("event", Event, model.HassEvent{})

	// testing
	/*go func() {
		ticker := time.NewTicker(3 * time.Second)
		for range ticker.C {
			app.GetHassCaller().LightToggle("light.escalier_switch")
		}
	}()*/

	app.Start()

	go func() {
		app.GetWebSocketClient().SubscribeEvents("state_changed")
		app.GetWebSocketClient().SubscribeEvents("roku_command")
	}()

	/*entity, err := app.GetSimpleClient().GetEntity("input_boolean", `override_[a-z]*_living`)
	if err != nil {
		log.Fatalf("An error occurred when getting entity, err=%v", err)
	}

	log.Printf("entity=%s, state=%s, domain=%s", entity.EntityID, entity.State.State, entity.Domain)
	*/

	test := module.NewFreeboxChecker(2*time.Second, "8.8.8.8", model.HassEntity{
		EntityID: "living_fbx",
		Domain:   "switch",
	})
	err := test.Start()
	if err != nil {
		log.Println("Error starting module FreeboxChecker")
	}

	// Main loop, ctrl+c to stop
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for range ticker.C {
		select {
		case <-interrupt:
			log.Println("Stopping service")
			select {
			case <-time.After(time.Second):
				app.GetWebSocketClient().Close()
			}
			return
		}
	}
}

// Event godoc
func Event(msg model.HassAPIObject) {
	//event := msg.(*model.HassEvent)
	//log.Printf("Received: event, msg=%+v\n", event)
}
