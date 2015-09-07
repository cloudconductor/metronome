package scheduler

import (
	"bufio"
	"bytes"
	"io"
	"metronome/config"
	"metronome/queue"
	"metronome/util"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/hashicorp/consul/api"
)

const TASK_TIMEOUT_WITHOUT_START = 120
const TASK_TIMEOUT = 3600

func (s *Scheduler) Run() {
	if err := s.getNode(); err != nil {
		panic(err)
	}

	ch := make(chan EventTask)
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

func (s *Scheduler) polling(ch chan EventTask) error {
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
			log.Debug(et.String())
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
		log.Debugf("Wait a task will have been finished by other instance(%s)", eventTasks[0].String())
	}
	return nil
}

//	Trigger channel when current task has been reached timeout
func taskTimeout(ch chan EventTask) {
	var prev EventTask
	var now EventTask
	for {
		time.Sleep(1 * time.Second)
		pq := &queue.Queue{
			Client: util.Consul(),
			Key:    PROGRESS_QUEUE_KEY,
		}

		//	Wait until task has dispatched
		if err := pq.FetchHead(&now); err != nil || now.ID == "" || prev.ID == now.ID && prev.No == now.No {
			continue
		}
		prev = now

		//	Wait until current task has started on any node or timeout
		cancel := make(chan bool)
		select {
		case <-startTask(now, cancel):
		case <-time.After(TASK_TIMEOUT_WITHOUT_START * time.Second):
			cancel <- true
			ch <- now
			continue
		}

		//	Wait until current task has finished or timeout
		select {
		case <-changeTask(now, cancel):
		case <-time.After(time.Duration(TASK_TIMEOUT) * time.Second):
			cancel <- true
			ch <- now
		}
	}
}

//	Trigger channel when change current task
func changeTask(et EventTask, cancel chan bool) chan bool {
	ch := make(chan bool)

	go func(chan bool) {
		pq := &queue.Queue{
			Client: util.Consul(),
			Key:    PROGRESS_QUEUE_KEY,
		}

		for {
			time.Sleep(1 * time.Second)
			//	Exit when cancel channel has signalled
			select {
			case <-cancel:
				return
			default:
			}

			//  Send signal to change task channel when current task has been changed
			var now EventTask
			if err := pq.FetchHead(&now); err != nil {
				ch <- true
				return
			}

			if et.ID != now.ID || et.No != now.No {
				ch <- true
				return
			}
		}
	}(ch)

	return ch
}

//	Trigger channel when start current task
func startTask(et EventTask, cancel chan bool) chan bool {
	ch := make(chan bool)
	go func(chan bool) {
		for {
			time.Sleep(1 * time.Second)
			//	Exit when cancel channel has signalled
			select {
			case <-cancel:
				return
			default:
			}

			//	Send signal to start task channel when task result has been written
			if r, err := getTaskResult(et.ID, et.No); err != nil || r != nil {
				ch <- true
				return
			}
		}
	}(ch)

	return ch
}

func (scheduler *Scheduler) getNode() error {
	var err error
	scheduler.node, err = os.Hostname()
	return err
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

	log.Infof("Run task(%s)", task.String())

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
	log.Infof("Finish task(%s)", task.String())
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
		if task.Skippable {
			status = "skip"
		} else {
			status = "timeout"
		}
	}

	for _, nr := range nodeResults {
		if nr.Status == "error" {
			status = "error"
			break
		}
		if nr.Status == "inprogress" {
			status = "timeout"
		}
	}

	if status == "error" || status == "timeout" {
		// remove following tasks in progress task queue when some error occured or task has reached timeout
		pq.Clear()
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
