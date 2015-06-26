package scheduler

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"path/filepath"
	"scheduler/config"
	"scheduler/task"
	"scheduler/util"
	"strings"
	"time"

	"github.com/ghodss/yaml"

	"github.com/hashicorp/consul/api"
)

type Scheduler struct {
	schedules map[string]Schedule
	node      string
}

type DispatchTask struct {
	pattern string
	task    task.Task
}

func NewScheduler() (*Scheduler, error) {
	scheduler := &Scheduler{}
	scheduler.schedules = make(map[string]Schedule)

	err := scheduler.load()
	if err != nil {
		return nil, err
	}

	scheduler.node, err = getSelfNode()
	if err != nil {
		return nil, err
	}
	fmt.Printf("Scheduler initialized(Self node: %s)\n", scheduler.node)

	return scheduler, nil
}

func (scheduler *Scheduler) Run() {
	eq := &Queue{Client: util.Consul(), Node: scheduler.node}

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
	entries, err := ioutil.ReadDir(filepath.Join(config.BaseDir, "patterns"))
	if err != nil {
		return err
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}

		path := filepath.Join(config.BaseDir, "patterns", e.Name(), "task.yml")
		if !util.Exists(path) {
			fmt.Printf("Schedule file does not found(%s)\n", path)
			continue
		}

		d, err := ioutil.ReadFile(path)
		if err != nil {
			return errors.New(fmt.Sprintf("Failed to load config file(%s)\n\t%s", path, err))
		}
		var schedule Schedule
		err = yaml.Unmarshal([]byte(d), &schedule)
		if err != nil {
			return errors.New(fmt.Sprintf("Failed to unmarshal json(%s)\n\t%s", path, err))
		}
		schedule.SetPattern(e.Name())
		scheduler.schedules[e.Name()] = schedule
	}
	return nil
}

func getSelfNode() (string, error) {
	if config.Node != "" {
		return config.Node, nil
	}

	var err error
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	nodes, _, err := util.Consul().Catalog().Nodes(&api.QueryOptions{})
	if err != nil {
		return "", err
	}
	for _, n := range nodes {
		for _, a := range addrs {
			h := strings.Split(a.String(), "/")[0]
			if n.Address == h {
				return n.Node, nil
			}
		}
	}

	return "", errors.New("Current system ip address does not found in consul catalog")
}

func (scheduler *Scheduler) dispatch(trigger string) error {
	tasks := scheduler.filter(trigger)
	if len(tasks) == 0 {
		return errors.New(fmt.Sprintf("Task %s is not defined", trigger))
	}
	for _, t := range tasks {
		if err := t.task.Run(scheduler.schedules[t.pattern].Variables); err != nil {
			return err
		}
	}

	return nil
}

func (scheduler *Scheduler) filter(trigger string) []DispatchTask {
	var tasks []DispatchTask
	for k, v := range scheduler.schedules {
		for _, t := range v.Tasks {
			if t.Trigger == trigger {
				tasks = append(tasks, DispatchTask{pattern: k, task: t})
			}
		}
	}
	return tasks
}

func Push(trigger string) (string, error) {
	var node string
	var err error
	node, err = getSelfNode()
	if err != nil {
		return "", err
	}

	eq := &Queue{Client: util.Consul(), Node: node}
	err = eq.EnQueue(Item{Type: trigger})
	if err != nil {
		return "", err
	}

	fmt.Printf("Push event to queue(Node: %s, Type: %s)\n", node, trigger)
	return "", nil
}
