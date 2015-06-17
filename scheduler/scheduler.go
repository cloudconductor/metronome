package scheduler

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"scheduler/config"
	"scheduler/task"
	"strings"
	"time"

	"github.com/ghodss/yaml"

	"github.com/hashicorp/consul/api"
)

type Scheduler struct {
	schedule Schedule
	client   *api.Client
	node     string
}

func NewScheduler() (*Scheduler, error) {
	scheduler := &Scheduler{}
	err := scheduler.load()
	if err != nil {
		return nil, err
	}

	err = scheduler.connect()
	if err != nil {
		return nil, err
	}

	return scheduler, nil
}

func (scheduler *Scheduler) Run() {
	eq := &Queue{Client: scheduler.client, Node: scheduler.node}

	for {
		fmt.Println(time.Now())
		item, err := eq.DeQueue()
		if err != nil {
			fmt.Println(err)
			return
		}
		if item != nil {
			fmt.Printf("Receive item %s\n", item.Type)
			err = scheduler.dispatch(item.Type)
			if err != nil {
				fmt.Println(err)
			}
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

	var err error
	scheduler.client, err = api.NewClient(consul)
	if err != nil {
		fmt.Println("Failed to create consul.Client")
		return err
	}

	if config.Node != "" {
		scheduler.node = config.Node
		return nil
	}

	fmt.Println("Node does not set, will search self ip address from consul catalog")
	scheduler.node, err = scheduler.findSelfNode()
	return err
}

func (scheduler *Scheduler) dispatch(trigger string) error {
	tasks := scheduler.filter(trigger)
	if len(tasks) == 0 {
		return errors.New(fmt.Sprintf("Task %s is not defined", trigger))
	}
	for _, t := range tasks {
		if err := t.Run(); err != nil {
			return err
		}
	}

	return nil
}

func (scheduler *Scheduler) filter(trigger string) []task.Task {
	var tasks []task.Task
	for _, t := range scheduler.schedule.Tasks {
		if t.Trigger == trigger {
			tasks = append(tasks, t)
		}
	}
	return tasks
}

func (scheduler *Scheduler) findSelfNode() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	nodes, _, err := scheduler.client.Catalog().Nodes(&api.QueryOptions{})
	if err != nil {
		return "", err
	}
	for _, n := range nodes {
		for _, a := range addrs {
			h := strings.Split(a.String(), "/")[0]
			fmt.Printf("Addresses = %s\t", h)
			fmt.Printf("Nodes= %s\n", n.Address)
			if n.Address == h {
				return n.Node, nil
			}
		}
	}

	return "", errors.New("Current system ip address does not found in consul catalog")
}
