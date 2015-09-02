package scheduler

import (
	"encoding/json"
	"errors"
	"fmt"
	"metronome/config"
	"metronome/util"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/hashicorp/consul/api"
)

type EventTask struct {
	Pattern   string
	ID        string
	No        int
	Service   string
	Tag       string
	Task      string
	Skippable bool
}

func (et *EventTask) UnmarshalJSON(d []byte) error {
	et.Skippable = config.Skippable

	m := make(map[string]json.RawMessage)
	u := &util.UnmarshalContext{}
	u.Unmarshal(d, &m)
	u.Unmarshal([]byte(m["pattern"]), &et.Pattern)
	u.Unmarshal([]byte(m["id"]), &et.ID)
	u.Unmarshal([]byte(m["no"]), &et.No)
	u.Unmarshal([]byte(m["service"]), &et.Service)
	u.Unmarshal([]byte(m["tag"]), &et.Tag)
	u.Unmarshal([]byte(m["task"]), &et.Task)
	u.Unmarshal([]byte(m["skippable"]), &et.Skippable)
	return u.Err
}

func (et EventTask) MarshalJSON() ([]byte, error) {
	et.Skippable = config.Skippable
	var columns []string
	columns = append(columns, fmt.Sprintf("\"pattern\": \"%s\"", et.Pattern))
	columns = append(columns, fmt.Sprintf("\"id\": \"%s\"", et.ID))
	columns = append(columns, fmt.Sprintf("\"no\": %d", et.No))
	columns = append(columns, fmt.Sprintf("\"service\": \"%s\"", et.Service))
	columns = append(columns, fmt.Sprintf("\"tag\": \"%s\"", et.Tag))
	columns = append(columns, fmt.Sprintf("\"task\": \"%s\"", et.Task))
	columns = append(columns, fmt.Sprintf("\"skippable\": %s", strconv.FormatBool(et.Skippable)))
	return []byte(fmt.Sprintf("{ %s }", strings.Join(columns, ","))), nil
}

func (et *EventTask) Runnable(node string) bool {
	//	Target node doesn't have conditional service or tag
	if !util.HasCatalogRecord(node, et.Service, et.Tag) {
		return false
	}

	//	Skip task when task had executed already
	nodeResult, err := getNodeTaskResult(et.ID, et.No, node)
	if err != nil {
		return false
	}
	if nodeResult != nil && nodeResult.IsFinished() {
		return false
	}
	return true
}

func (et *EventTask) IsFinished(ch chan bool) bool {
	//	Finished task when timeout has occurred
	select {
	case <-ch:
		return true
	default:
	}

	nodes, _, err := util.Consul().Catalog().Nodes(&api.QueryOptions{})
	if err != nil {
		return false
	}

	filteredNodes := et.filterNodes(nodes)
	if len(filteredNodes) == 0 {
		if et.Skippable {
			log.Warnf("Skip task(Task: %s, ID: %s, No: %d, Service: %s, Tag: %s)", et.Task, et.ID, et.No, et.Service, et.Tag)
			return true
		}
		return false
	}

	for _, node := range filteredNodes {
		result, err := getNodeTaskResult(et.ID, et.No, node.Node)
		if err != nil || result == nil || !result.IsFinished() {
			return false
		}
	}
	return true
}

//	Filter nodes by conditional service and tag
func (et *EventTask) filterNodes(nodes []*api.Node) []*api.Node {
	var results []*api.Node
	for _, node := range nodes {
		r, err := getNodeTaskResult(et.ID, et.No, node.Node)
		if err == nil && r != nil || util.HasCatalogRecord(node.Node, et.Service, et.Tag) {
			results = append(results, node)
		}
	}
	return results
}

//	Run operations in task
func (et *EventTask) Run(scheduler *Scheduler) error {
	t, found := scheduler.schedules[et.Pattern].Tasks[et.Task]
	if !found {
		return errors.New(fmt.Sprintf("Target task(%s) does not defined in %s\n", et.Task, et.Pattern))
	} else {
		return t.Run(scheduler.schedules[t.Pattern].Variables)
	}
}

func (et *EventTask) GetResult() (*TaskResult, error) {
	result, err := getTaskResult(et.ID, et.No)
	if err != nil {
		return nil, err
	}

	if result == nil {
		result = &TaskResult{
			EventID:   et.ID,
			No:        et.No,
			Name:      et.Task,
			Status:    "inprogress",
			StartedAt: time.Now(),
		}
	}
	return result, nil
}

func (et *EventTask) WriteStartLog(node string) error {
	//	Log starting task as TaskResult on KVS
	result, err := getTaskResult(et.ID, et.No)
	if err != nil {
		return err
	}
	if result == nil {
		result = &TaskResult{
			EventID:   et.ID,
			No:        et.No,
			Name:      et.Task,
			Status:    "inprogress",
			StartedAt: time.Now(),
		}
		if err := result.Save(); err != nil {
			return err
		}
	}

	//	Log starting task on node as NodeTaskResult on KVS
	nodeResult := &NodeTaskResult{
		EventID:   et.ID,
		No:        et.No,
		Node:      node,
		Status:    "inprogress",
		StartedAt: time.Now(),
	}
	return nodeResult.Save()
}

func (et *EventTask) WriteFinishLog(node string, status string, log string) error {
	//	Log finishing task on node as NodeTaskResult on KVS
	nodeResult, err := getNodeTaskResult(et.ID, et.No, node)
	if err != nil {
		return err
	}
	nodeResult.FinishedAt = time.Now()
	nodeResult.Status = status
	nodeResult.Log = log

	return nodeResult.Save()
}
