package scheduler

import (
	"errors"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/ghodss/yaml"
)

type Scheduler struct {
	config Config
}

func NewScheduler(path string) (*Scheduler, error) {
	scheduler := &Scheduler{}
	err := scheduler.load(path)
	if err != nil {
		return nil, err
	}

	return scheduler, nil
}

func (scheduler *Scheduler) Run() {
	for {
		fmt.Println(time.Now())
		time.Sleep(1 * time.Second)
	}
}

func (scheduler *Scheduler) load(path string) error {
	d, err := ioutil.ReadFile(path)
	if err != nil {
		return errors.New(fmt.Sprintf("Failed to load config file(%s)", path))
	}
	return yaml.Unmarshal([]byte(d), &scheduler.config)
}
