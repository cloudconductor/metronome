package scheduler

import (
	"errors"
	"fmt"
	"scheduler/util"
	"time"

	"github.com/hashicorp/consul/api"
)

type EventTask struct {
	Pattern string
	ID      string
	No      int
	Service string
	Tag     string
	Task    string
}

func (et *EventTask) Runnable(node string) bool {
	if !util.HasCatalogRecord(node, et.Service, et.Tag) {
		return false
	}

	nodeResult, err := getNodeTaskResult(et.ID, et.No, node)
	if err != nil {
		return false
	}
	if nodeResult != nil && nodeResult.IsFinished() {
		return false
	}
	return true
}

func (et *EventTask) IsFinished() bool {
	nodes, _, err := util.Consul().Catalog().Nodes(&api.QueryOptions{})
	if err != nil {
		return false
	}

	for _, node := range nodes {
		if util.HasCatalogRecord(node.Node, et.Service, et.Tag) {
			result, err := getNodeTaskResult(et.ID, et.No, node.Node)
			if err != nil || result == nil || !result.IsFinished() {
				return false
			}
		}
	}
	return true
}

func (et *EventTask) Run(scheduler *Scheduler) error {
	//	Run operations in task
	t, found := scheduler.schedules[et.Pattern].Tasks[et.Task]
	if !found {
		return errors.New(fmt.Sprintf("Target task(%s) does not defined in %s\n", et.Task, et.Pattern))
	} else {
		return t.Run(scheduler.schedules[t.Pattern].Variables)
	}
}

func (et *EventTask) WriteStartLog(node string) error {
	//	Log starting task as TaskResult on KVS
	result, err := getTaskResult(et.ID, et.No)
	if err != nil {
		return err
	}
	if result == nil {
		result = &TaskResult{EventID: et.ID, No: et.No, Name: et.Task, Status: "inprogress", StartedAt: time.Now()}
		if err := result.Save(); err != nil {
			return err
		}
	}

	//	Log starting task on node as NodeTaskResult on KVS
	nodeResult := &NodeTaskResult{EventID: et.ID, No: et.No, Node: node, Status: "inprogress", StartedAt: time.Now()}
	return nodeResult.Save()
}

func (et *EventTask) WriteFinishLog(node string, status string) error {
	//	Log finishing task on node as NodeTaskResult on KVS
	nodeResult, err := getNodeTaskResult(et.ID, et.No, node)
	if err != nil {
		return err
	}
	nodeResult.FinishedAt = time.Now()
	nodeResult.Status = status

	return nodeResult.Save()
}
