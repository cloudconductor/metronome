package task

import (
	"encoding/json"
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

type unmarshalContext struct {
	err error
}

func (u *unmarshalContext) unmarshal(data []byte, v interface{}) error {
	if u.err != nil {
		return u.err
	}
	u.err = json.Unmarshal(data, v)
	return u.err
}

func (u *unmarshalContext) unmarshalOperations(data []byte, v *[]operation.Operation) error {
	if u.err != nil {
		return u.err
	}
	u.err = operation.UnmarshalOperations(data, v)
	return u.err
}

func (t *Task) UnmarshalJSON(d []byte) error {
	m := make(map[string]json.RawMessage)
	u := &unmarshalContext{}
	u.unmarshal(d, &m)
	u.unmarshal([]byte(m["name"]), &t.Name)
	u.unmarshal([]byte(m["trigger"]), &t.Trigger)
	u.unmarshal([]byte(m["description"]), &t.Description)
	u.unmarshal([]byte(m["filter"]), &t.Filter)
	u.unmarshalOperations([]byte(m["operations"]), &t.Operations)

	fmt.Printf("Loaded %v\n", t)
	return u.err
}

func (t *Task) Run() error {
	fmt.Printf("Task %s has started\n", t.Name)
	for _, o := range t.Operations {
		err := o.Run()
		if err != nil {
			return err
		}
	}
	fmt.Printf("Task %s has finished\n", t.Name)
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
