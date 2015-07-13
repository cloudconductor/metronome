package operation

import (
	"encoding/json"
	"fmt"
	"scheduler/queue"
	"scheduler/util"

	"github.com/hashicorp/consul/api"
)

type TaskOperation struct {
	BaseOperation
	Name   string
	Filter struct {
		Service string
		Tag     string
	}
	Parameters map[string]string
}

func NewTaskOperation(v json.RawMessage) *TaskOperation {
	o := &TaskOperation{}
	json.Unmarshal(v, &o)
	return o
}

func (o *TaskOperation) Run(vars map[string]string) error {
	nodes, _, err := util.Consul().Catalog().Nodes(&api.QueryOptions{})
	if err != nil {
		return err
	}

	for _, node := range nodes {
		if !util.HasCatalogRecord(node.Node, o.Filter.Service, o.Filter.Tag) {
			continue
		}

		eq := &queue.Queue{Client: util.Consul(), Key: "task_queue" + node.Node}
		err = eq.EnQueue(queue.TaskEvent{Name: o.Name})
		if err != nil {
			return err
		}

		fmt.Printf("Enqueue %s on %s\n", o.Name, node.Node)
	}
	return nil
}

func (o *TaskOperation) String() string {
	return "task"
}
