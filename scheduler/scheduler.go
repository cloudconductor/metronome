package scheduler

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"scheduler/config"
	"scheduler/task"
	"time"

	"github.com/ghodss/yaml"

	"github.com/hashicorp/consul/api"
)

type Scheduler struct {
	schedule Schedule
	client   *api.Client
}

func NewScheduler() (*Scheduler, error) {
	scheduler := &Scheduler{}
	err := scheduler.load()
	if err != nil {
		return nil, err
	}

	err = scheduler.connect()
	if err != nil {
		fmt.Println("Failed to create consul.Client")
		return nil, err
	}

	return scheduler, nil
}

func (scheduler *Scheduler) Run() {
	eq := &Queue{Client: scheduler.client, Node: "dummy"}

	for {
		fmt.Println(time.Now())
		item, err := eq.DeQueue()
		if err != nil {
			fmt.Println(err)
			return
		}
		if item != nil {
			err = scheduler.dispatch(item.Type)
			if err != nil {
				fmt.Println(err)
			}
			fmt.Printf("Receive item %s\n", item.Type)
		}
		time.Sleep(1 * time.Second)
	}
}

func (scheduler *Scheduler) load() error {
	path := config.ScheduleFile
	d, err := ioutil.ReadFile(path)
	if err != nil {
		return errors.New(fmt.Sprintf("Failed to load config file(%s)", path))
	}
	return yaml.Unmarshal([]byte(d), &scheduler.schedule)
}

func (scheduler *Scheduler) connect() error {
	consul := api.DefaultConfig()
	consul.Token = config.Token
	consul.Address = fmt.Sprintf("%s:%d", config.Hostname, config.Port)
	consul.Scheme = config.Protocol
	consul.HttpClient.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: config.InsecureSkipVerify,
		},
	}

	client, err := api.NewClient(consul)
	scheduler.client = client
	return err
}

func (scheduler *Scheduler) dispatch(trigger string) error {
	task, found := scheduler.find(trigger)
	if !found {
		return errors.New(fmt.Sprintf("Task %s is not defined", trigger))
	}

	return task.Run()
}

func (scheduler *Scheduler) find(trigger string) (*task.Task, bool) {
	for _, t := range scheduler.schedule.Tasks {
		if t.Trigger == trigger {
			return &t, true
		}
	}
	return nil, false
}
