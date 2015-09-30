package main

import (
	"flag"
	"fmt"
	"metronome/scheduler"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/takama/daemon"
)

type Service struct {
	daemon.Daemon
}

func NewService(name, description string, dependencies []string) (*Service, error) {
	srv, err := daemon.New(name, description, dependencies)
	if err != nil {
		return nil, err
	}

	return &Service{srv}, nil
}

func (service *Service) Manage() (string, error) {
	usage := "Usage: metronome install | remove | start | stop | status | agent"

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
		case "agent":
			return agent()
		case "push":
			return scheduler.Push()
		case "dispatch":
			return dispatch(flag.Args()[1])
		case "version":
			return fmt.Sprintf("metronome %s\n", Version), nil
		default:
			return usage, nil
		}
	}
	return agent()
}

func agent() (string, error) {
	time.Sleep(5 * time.Second)
	scheduler, err := scheduler.NewScheduler()
	if err != nil {
		return "Failed to create scheduler", err
	}
	go scheduler.Run()

	return waitSignal()
}

func waitSignal() (string, error) {
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

func dispatch(trigger string) (string, error) {
	scheduler, err := scheduler.NewScheduler()
	if err != nil {
		return "Failed to create scheduler", err
	}
	if err := scheduler.Dispatch(trigger); err != nil {
		return fmt.Sprintf("Failed to dispatch event(%s)", trigger), err
	}
	return "", nil
}
