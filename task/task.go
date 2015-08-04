package task

import (
	"encoding/json"
	"errors"
	"fmt"
	"scheduler/operation"
	"time"

	log "github.com/Sirupsen/logrus"
)

type Task struct {
	Pattern     string
	Name        string
	Trigger     string
	Description string
	Timeout     int32
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
	if u.err != nil || len(data) == 0 {
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
	u.unmarshal([]byte(m["timeout"]), &t.Timeout)
	if _, ok := m["filter"]; ok {
		u.unmarshal([]byte(m["filter"]), &t.Filter)
	}
	u.unmarshalOperations([]byte(m["operations"]), &t.Operations)

	return u.err
}

func (t *Task) SetPattern(pattern string) {
	t.Pattern = pattern
	for _, o := range t.Operations {
		o.SetPattern(pattern)
	}
}

func (t *Task) Run(vars map[string]string) error {
	log.Infof("-- Task %s has started", t.Name)
	ch := make(chan error)
	timeout := make(chan bool)

	go t.runWithTimeout(vars, ch, timeout)

	select {
	case err := <-ch:
		if err != nil {
			log.Errorf("-- Task %s has failed", t.Name)
			return err
		}
	case <-time.After(time.Duration(t.Timeout) * time.Second):
		log.Errorf("-- Task %s has expired", t.Name)
		close(timeout)
		return errors.New("Timeout expired while executing task")
	}
	log.Infof("-- Task %s has finished successfully", t.Name)
	return nil
}

func (t *Task) runWithTimeout(vars map[string]string, ch chan error, timeout <-chan bool) {
	for _, o := range t.Operations {
		log.Infof("---- Operation %s has started", o.String())
		if err := o.Run(vars); err != nil {
			log.Errorf("---- Operation %s in %s has failed", o.String(), t.Name)
			ch <- err
			return
		}

		select {
		case <-timeout:
			return
		default:
			log.Infof("---- Operation %s has finished successfully", o.String())
		}
	}
	ch <- nil
}

func (t *Task) String() string {
	var s string

	s += fmt.Sprintf("Task %s\n", t.Name)
	s += fmt.Sprintf("  Name: %s\n", t.Name)
	s += fmt.Sprintf("  Pattern: %s\n", t.Pattern)
	s += fmt.Sprintf("  Trigger: %s\n", t.Trigger)
	s += fmt.Sprintf("  Description: %s\n", t.Description)
	s += fmt.Sprintf("  Timeout: %d\n", t.Timeout)
	s += fmt.Sprintf("  Filter: %v\n", t.Filter)

	s += "  Operations:\n"
	for _, o := range t.Operations {
		s += fmt.Sprintf("    %v\n", o)
	}
	return s
}
