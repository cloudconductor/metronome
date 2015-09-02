package scheduler

import (
	"errors"
	"fmt"
)

//	Dispatch event immediately when execute metronome with dispatch subcommand
func (scheduler *Scheduler) Dispatch(name string) error {
	events := scheduler.sortedEvents(name)
	if len(events) == 0 {
		return errors.New(fmt.Sprintf("Event %s is not defined", name))
	}

	for _, e := range events {
		if err := e.Run(scheduler); err != nil {
			return err
		}
	}

	return nil
}
