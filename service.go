package main

import (
	"fmt"
	"os"
	"os/signal"
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

func (service *Service) New() {
	fmt.Println("service: init")
}

func (service *Service) Manage() (string, error) {
	usage := "Usage: scheduler install | remove | start | stop | status"

	if len(os.Args) > 1 {
		switch os.Args[1] {
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

	scheduler, err := NewScheduler()
	if err != nil {
		return "Failed to create scheduler", err
	}
	go scheduler.Run()

	return service.WaitSignal()
}

func (service *Service) WaitSignal() (string, error) {
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
