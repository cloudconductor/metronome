package scheduler

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/ghodss/yaml"

	"github.com/hashicorp/consul/api"
)

type Scheduler struct {
	schedule Schedule
	client   *api.Client
}

func NewScheduler(path string, config *Config) (*Scheduler, error) {
	scheduler := &Scheduler{}
	err := scheduler.load(path)
	if err != nil {
		return nil, err
	}

	err = scheduler.connect(config)
	if err != nil {
		fmt.Println("Failed to create consul.Client")
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
	return yaml.Unmarshal([]byte(d), &scheduler.schedule)
}

func (scheduler *Scheduler) connect(config *Config) error {
	consul := api.DefaultConfig()
	consul.Address = fmt.Sprintf("%s:%d", config.Hostname, config.Port)
	consul.Token = config.Token
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
