package main

import (
	"metronome/config"
	"metronome/util"
	"os"

	log "github.com/Sirupsen/logrus"
)

func main() {
	log.SetFormatter(&util.LogFormatter{})
	if config.Debug {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	service, err := NewService("metronome", "Scheduling tool on consul", []string{"consul.service"})
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	status, err := service.Manage()
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	log.Info(status)
}
