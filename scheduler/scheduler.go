package scheduler

import (
	"fmt"
	"time"
)

type Scheduler struct {
}

func NewScheduler(path string) (*Scheduler, error) {
	scheduler := &Scheduler{}

	return scheduler, nil
}

func (scheduler *Scheduler) Run() {
	for {
		fmt.Println(time.Now())
		time.Sleep(1 * time.Second)
	}
}
