package task

import (
	"encoding/json"
	"fmt"
	"os"
	"scheduler/operation"
	"scheduler/util"

	"github.com/hashicorp/consul/api"
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
	if !t.canRun() {
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

func (t *Task) canRun() bool {
	if t.Filter.Service == "" && t.Filter.Tag == "" {
		return true
	}

	node, err := os.Hostname()
	if err != nil {
		return false
	}

	catalog, _, err := util.Consul().Catalog().Node(node, &api.QueryOptions{})
	if err != nil {
		return false
	}

	service, ok := catalog.Services[t.Filter.Service]
	if !ok {
		return false
	}

	if t.Filter.Tag == "" {
		return true
	}

	for _, s := range service.Tags {
		if s == t.Filter.Tag {
			return true
		}
	}

	return false
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
