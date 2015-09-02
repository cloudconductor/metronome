package scheduler

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"metronome/config"
	"metronome/queue"
	"metronome/util"
	"os"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/hashicorp/consul/api"
)

const TASK_TIMEOUT = 1800

func (s *Scheduler) Run() {
	if err := s.connect(); err != nil {
		panic(err)
	}

	ch := make(chan bool)
	go taskTimeout(ch)
	for {
		time.Sleep(1 * time.Second)

		if config.Debug {
			log.Debug(time.Now())
			log.Debug("Wait at before polling until enter key has been pressed")
			scanner := bufio.NewScanner(os.Stdin)
			scanner.Scan()
		}

		if err := s.polling(ch); err != nil {
			log.Error(err)
			continue
		}
	}
}

func (s *Scheduler) polling(ch chan bool) error {
	//	Create critical section by consul lock
	l, err := util.Consul().LockKey(LOCK_KEY)
	if err != nil {
		return err
	}
	if _, err := l.Lock(nil); err != nil {
		return err
	}
	defer l.Unlock()

	//	Polling tasks from queue
	var eventTasks []EventTask
	pq := &queue.Queue{
		Client: util.Consul(),
		Key:    PROGRESS_QUEUE_KEY,
	}
	if err := pq.Items(&eventTasks); err != nil {
		return err
	}

	if config.Debug {
		log.Debug("-------- Progress Task Queue --------")
		nodes, _, _ := util.Consul().Catalog().Nodes(&api.QueryOptions{})
		for _, et := range eventTasks {
			log.Debugf("Task: %s, %s, %s", et.Task, et.Service, et.Tag)
			for _, n := range nodes {
				log.Debugf("%s: %t", n.Node, et.Runnable(n.Node))
			}
		}
	}

	switch {
	case len(eventTasks) == 0:
		return s.dispatchEvent()
	case eventTasks[0].Runnable(s.node):
		//	runTask is parallelizable
		l.Unlock()
		return s.runTask(eventTasks[0])
	case eventTasks[0].IsFinished(ch):
		return s.finishTask(eventTasks[0])
	default:
		log.Debugf("Wait a task will have been finished by other instance(Task: %s, Service: %s, Tag: %s)", eventTasks[0].Task, eventTasks[0].Service, eventTasks[0].Tag)
	}
	return nil
}

//	Trigger channel when current task has been reached timeout
func taskTimeout(ch chan bool) {
	for {
		select {
		case <-changeTask():
		case <-startTask():
		case <-time.After(time.Duration(TASK_TIMEOUT) * time.Second):
			ch <- true
		}
	}
}

//	Trigger channel when change current task
func changeTask() chan bool {
	ch := make(chan bool)

	go func(chan bool) {
		pq := &queue.Queue{
			Client: util.Consul(),
			Key:    PROGRESS_QUEUE_KEY,
		}

		var prev EventTask
		if err := pq.FetchHead(&prev); err != nil {
			ch <- true
			return
		}

		for {
			time.Sleep(1 * time.Second)
			var now EventTask
			if err := pq.FetchHead(&now); err != nil {
				ch <- true
				return
			}

			if prev.ID != now.ID || prev.No != now.No {
				ch <- true
				return
			}
		}
	}(ch)

	return ch
}

//	Trigger channel when start current task
func startTask() chan bool {
	ch := make(chan bool)
	go func(chan bool) {
		pq := &queue.Queue{
			Client: util.Consul(),
			Key:    PROGRESS_QUEUE_KEY,
		}

		for {
			time.Sleep(1 * time.Second)
			var et EventTask
			if err := pq.FetchHead(&et); err != nil {
				ch <- true
				return
			}

			if r, err := getEventResult(et.ID); err != nil || r != nil {
				ch <- true
				return
			}
		}
	}(ch)

	return ch
}

func (scheduler *Scheduler) connect() error {
	var err error
	scheduler.node, err = os.Hostname()
	if err != nil {
		return err
	}

	return scheduler.registerServer()
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

	if _, err := c.KV().Put(kv, &api.WriteOptions{}); err != nil {
		return err
	}

	return nil
}

//	Convert node name to address based on consul catalog
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

func (s *Scheduler) dispatchEvent() error {
	pq := &queue.Queue{
		Client: util.Consul(),
		Key:    PROGRESS_QUEUE_KEY,
	}
	eq := &queue.Queue{
		Client: util.Consul(),
		Key:    EVENT_QUEUE_KEY,
	}

	var consulEvent api.UserEvent
	if err, found := eq.DeQueue(&consulEvent); err != nil || !found {
		return err
	}
	result, err := getEventResult(consulEvent.ID)
	if err != nil {
		return err
	}
	if result != nil {
		log.Debugf("Ignore event(ID: %s, Name: %s) already has been executed", consulEvent.ID, consulEvent.Name)
		return nil
	}

	//	Collect events over all task.yml and dispatch tasks to progress task queue
	log.Infof("Dispatch event(ID: %s, Name: %s)", consulEvent.ID, consulEvent.Name)
	events := s.sortedEvents(consulEvent.Name)
	c := 0
	for _, v := range events {
		switch {
		case v.Task != "":
			pq.EnQueue(EventTask{
				Pattern:   v.Pattern,
				ID:        consulEvent.ID,
				No:        c,
				Task:      v.Task,
				Skippable: config.Skippable,
			})
			c += 1
		case len(v.OrderedTasks) > 0:
			for _, t := range v.OrderedTasks {
				t.Pattern = v.Pattern
				t.ID = consulEvent.ID
				t.No = c
				pq.EnQueue(t)
				c += 1
			}
		}
	}

	//	Log starting event as EventResult on KVS
	result = &EventResult{
		ID:        consulEvent.ID,
		Name:      consulEvent.Name,
		Status:    "inprogress",
		StartedAt: time.Now(),
	}
	return result.Save()
}

func (s *Scheduler) runTask(task EventTask) error {
	var b bytes.Buffer
	writer := io.MultiWriter(&b, os.Stdout)
	log.SetOutput(writer)

	log.Infof("Run task(Task: %s, ID: %s, No: %d, Service: %s, Tag: %s)", task.Task, task.ID, task.No, task.Service, task.Tag)

	//	Run single task with result log
	if err := task.WriteStartLog(s.node); err != nil {
		return err
	}

	status := "success"
	if err := task.Run(s); err != nil {
		status = "error"
		log.Error("Following error has occurred while executing task")
		log.Error(err)
	}

	return task.WriteFinishLog(s.node, status, b.String())
}

//	Finish current task when no node in consul catalog will execute current task
func (s *Scheduler) finishTask(task EventTask) error {
	log.Infof("Finish task(Task: %s, ID: %s, No: %d, Service: %s, Tag: %s)", task.Task, task.ID, task.No, task.Service, task.Tag)
	pq := &queue.Queue{
		Client: util.Consul(),
		Key:    PROGRESS_QUEUE_KEY,
	}

	result, err := task.GetResult()
	if err != nil {
		return err
	}

	nodeResults, err := result.GetNodeResults()
	if err != nil {
		return err
	}

	//	Collect task results over all nodes
	status := "success"
	if len(nodeResults) == 0 {
		status = "skip"
	}
	for _, nr := range nodeResults {
		if nr.Status == "error" {
			status = "error"
			// remove following tasks in progress task queue when some error has been occurred
			pq.Clear()
			break
		}
		if nr.Status == "inprogress" {
			status = "timeout"
			// remove following tasks in progress task queue when timeout has been occurred
			pq.Clear()
		}
	}

	//	Log finishing task as TaskResult on KVS
	result.Status = status
	result.FinishedAt = time.Now()
	if err := result.Save(); err != nil {
		return err
	}

	//	Dequeue task from task queue when finished task over all all nodes
	var dummy EventTask
	if err, _ := pq.DeQueue(&dummy); err != nil {
		return err
	}

	//	Log finishing event as EventResult on KVS when finished all task in a progress task queue
	var tasks []EventTask
	if err := pq.Items(&tasks); err != nil {
		return err
	}
	if len(tasks) == 0 {
		eventResult, err := getEventResult(task.ID)
		if err != nil {
			return err
		}
		eventResult.Status = status
		eventResult.FinishedAt = time.Now()
		if err := eventResult.Save(); err != nil {
			return err
		}
	}

	return nil
}
