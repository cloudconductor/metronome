package task

import (
	"encoding/json"
	"errors"
	"fmt"
	"scheduler/operation"
)

type Task struct {
	Name        string
	Trigger     string
	Description string
	Filter      Filter
	Operations  []operation.Operation
}

type Filter struct {
	Service string
	Tag     string
}

func (t *Task) UnmarshalJSON(d []byte) error {
	m := make(map[string]json.RawMessage)
	err := json.Unmarshal(d, &m)
	if err != nil {
		return err
	}

	t.Name = string(m["name"])
	t.Trigger = string(m["trigger"])
	t.Description = string(m["description"])
	err = json.Unmarshal([]byte(m["filter"]), &t.Filter)
	if err != nil {
		return err
	}

	err = t.unmarshalOperations([]byte(m["operations"]))
	return err
}

func (t *Task) unmarshalOperations(d []byte) error {
	var list []map[string]json.RawMessage
	err := json.Unmarshal(d, &list)
	if err != nil {
		return err
	}

	for _, m := range list {
		if len(m) != 1 {
			return errors.New("Operation has multiple types")
		}

		for k, v := range m {
			factory, ok := operation.Operations[k]
			if !ok {
				return errors.New(fmt.Sprintf("Operation %s is not defined", k))
			}
			operation, err := factory(v)
			if err != nil {
				return err
			}
			t.Operations = append(t.Operations, operation)
		}
	}
	return nil
}

func (t *Task) Run() error {
	for _, o := range t.Operations {
		err := o.Run()
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *Task) String() string {
	var s string

	s += fmt.Sprintf("Task %s\n", t.Name)
	s += fmt.Sprintf("  Trigger: %s\n", t.Trigger)
	s += fmt.Sprintf("  Description: %s\n", t.Description)
	s += fmt.Sprintf("  Filter: %v\n", t.Filter)

	s += "  Operations:\n"
	for _, o := range t.Operations {
		s += fmt.Sprintf("    %s\n", o.Name())
	}
	return s
}
