package operation

import "encoding/json"

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
	return nil
}

func (o *TaskOperation) String() string {
	return "task"
}
