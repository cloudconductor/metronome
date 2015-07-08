package task

import (
	"encoding/json"
	"fmt"
	"os"
	"scheduler/operation"
	"scheduler/util"
)

type Task struct {
	pattern     string
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
	if _, ok := m["filter"]; ok {
		u.unmarshal([]byte(m["filter"]), &t.Filter)
	}
	u.unmarshalOperations([]byte(m["operations"]), &t.Operations)

	fmt.Printf("Loaded %v\n", t)
	return u.err
}

func (t *Task) SetPattern(pattern string) {
	t.pattern = pattern
	for _, o := range t.Operations {
		o.SetPattern(pattern)
	}
}

func (t *Task) Run(vars map[string]string) error {
	node, err := os.Hostname()
	if err != nil {
		return err
	}

	if !util.HasCatalogRecord(node, t.Filter.Service, t.Filter.Tag) {
		fmt.Printf("Ignore task %s\n", t.Name)
		return nil
	}
	fmt.Printf("Task %s has started\n", t.Name)
	for _, o := range t.Operations {
		err := o.Run(vars)
		if err != nil {
			fmt.Printf("Task %s has failed\n", t.Name)
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
		s += fmt.Sprintf("    %v\n", o)
	}
	return s
}
