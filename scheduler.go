package main

import (
	"fmt"
	"time"
)

type Scheduler struct {
}

func NewScheduler() (*Scheduler, error) {
	return &Scheduler{}, nil
}

func (scheduler *Scheduler) Run() {
	for {
		fmt.Println(time.Now())
		time.Sleep(1 * time.Second)
	}
}
