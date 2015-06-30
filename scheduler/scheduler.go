package scheduler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
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

	fmt.Println("Scheduler initialized")
	return scheduler, nil
}

func (scheduler *Scheduler) Run() {
	err := scheduler.connect()
	if err != nil {
		panic(err)
	}

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

func (scheduler *Scheduler) connect() error {
	var err error
	scheduler.node, err = os.Hostname()
	if err != nil {
		return err
	}

	return scheduler.registerServer()
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
	node, err = os.Hostname()
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

func (s *Scheduler) registerServer() error {
	var key = "cloudconductor/servers/" + s.node
	var c *api.Client = util.Consul()
	kv, _, err := c.KV().Get(key, &api.QueryOptions{})
	if err != nil {
		return err
	}

	if kv == nil {
		kv = &api.KVPair{Key: key}
	}

	m := make(map[string]interface{})
	m["roles"] = strings.Split(config.Role, ",")
	m["private_ip"], err = getAddress(s.node)
	if err != nil {
		return err
	}

	kv.Value, err = json.Marshal(m)
	if err != nil {
		return err
	}

	_, err = c.KV().Put(kv, &api.WriteOptions{})
	if err != nil {
		return err
	}

	return nil

}

func getAddress(node string) (string, error) {
	nodes, _, err := util.Consul().Catalog().Nodes(&api.QueryOptions{})
	if err != nil {
		return "", err
	}
	for _, n := range nodes {
		if n.Node == node {
			return n.Address, nil
		}
	}

	return "", errors.New("Current node does not found in consul catalog")
}
