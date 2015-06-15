package main

import (
	"flag"
	"os"
	"os/signal"
	"scheduler/scheduler"
	"syscall"

	"github.com/takama/daemon"
)

type Service struct {
	daemon.Daemon
}

func NewService(name, description string) (*Service, error) {
	srv, err := daemon.New(name, description)
	if err != nil {
		return nil, err
	}

	return &Service{srv}, nil
}

func (service *Service) Manage() (string, error) {
	usage := "Usage: scheduler install | remove | start | stop | status"

	config := &scheduler.Config{}
	config.Load()

	if flag.NArg() > 0 {
		switch flag.Args()[0] {
		case "install":
			return service.Install()
		case "remove":
			return service.Remove()
		case "start":
			return service.Start()
		case "stop":
			return service.Stop()
		case "status":
			return service.Status()
		default:
			return usage, nil
		}
	}

	scheduler, err := scheduler.NewScheduler("sample.yml", config)
	if err != nil {
		return "Failed to create scheduler", err
	}
	go scheduler.Run()

	return service.waitSignal()
}

func (service *Service) waitSignal() (string, error) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case killSignal := <-interrupt:
			if killSignal == os.Interrupt {
				return "Daemon was interrupted by system signal", nil
			}
			return "Daemon was killed", nil
		}
	}

	return "", nil
}
