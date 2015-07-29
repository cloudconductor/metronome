package scheduler

import (
	"encoding/json"
	"scheduler/util"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/consul/api"
)

const EVENT_RESULT_KEY = "scheduler/results"

type Result interface {
	Key() string
}

type EventResult struct {
	ID         string
	Name       string
	Status     string
	StartedAt  time.Time
	FinishedAt time.Time
}

type TaskResult struct {
	EventID    string
	No         int
	Name       string
	Status     string
	StartedAt  time.Time
	FinishedAt time.Time
}

type NodeTaskResult struct {
	EventID    string
	No         int
	Node       string
	Status     string
	StartedAt  time.Time
	FinishedAt time.Time
}

func (r *EventResult) Key() string {
	return EVENT_RESULT_KEY + "/" + r.ID
}

func (r *TaskResult) Key() string {
	return EVENT_RESULT_KEY + "/" + r.EventID + "/" + strconv.Itoa(r.No)
}

func (r *NodeTaskResult) Key() string {
	return EVENT_RESULT_KEY + "/" + r.EventID + "/" + strconv.Itoa(r.No) + "/" + r.Node
}

func (r *EventResult) Save() error {
	return putResult(r)
}

func (r *TaskResult) Save() error {
	return putResult(r)
}

func (r *NodeTaskResult) Save() error {
	return putResult(r)
}

func (r *EventResult) IsFinished() bool {
	return r.Status == "success" || r.Status == "error"
}

func (r *TaskResult) IsFinished() bool {
	return r.Status == "success" || r.Status == "error"
}

func (r *NodeTaskResult) IsFinished() bool {
	return r.Status == "success" || r.Status == "error"
}

func (r *TaskResult) GetNodeResults() ([]NodeTaskResult, error) {
	var results []NodeTaskResult

	prefix := EVENT_RESULT_KEY + "/" + r.EventID + "/" + strconv.Itoa(r.No)
	kvs, _, err := util.Consul().KV().List(prefix, &api.QueryOptions{})
	if err != nil {
		return nil, err
	}

	for _, kv := range kvs {
		node := strings.TrimPrefix(kv.Key, prefix)
		if node == "" {
			continue
		}
		result, err := getNodeTaskResult(r.EventID, r.No, node)
		if err != nil {
			return nil, err
		}
		results = append(results, *result)
	}
	return results, nil
}

func getEventResult(id string) (*EventResult, error) {
	var result EventResult
	key := EVENT_RESULT_KEY + "/" + id
	found, err := getResult(key, &result)
	if !found || err != nil {
		return nil, err
	}
	return &result, err
}

func getTaskResult(id string, no int) (*TaskResult, error) {
	var result TaskResult
	key := EVENT_RESULT_KEY + "/" + id + "/" + strconv.Itoa(no)
	found, err := getResult(key, &result)
	if !found || err != nil {
		return nil, err
	}
	return &result, err
}

func getNodeTaskResult(id string, no int, node string) (*NodeTaskResult, error) {
	var result NodeTaskResult
	key := EVENT_RESULT_KEY + "/" + id + "/" + strconv.Itoa(no) + "/" + node
	found, err := getResult(key, &result)
	if !found || err != nil {
		return nil, err
	}
	return &result, err
}

func getResult(key string, result interface{}) (bool, error) {
	kv, _, err := util.Consul().KV().Get(key, &api.QueryOptions{})
	if err != nil {
		return false, err
	}

	if kv == nil || len(kv.Value) == 0 {
		return false, nil
	}

	err = json.Unmarshal(kv.Value, &result)
	if err != nil {
		return false, err
	}
	return true, nil
}

func putResult(result Result) error {
	d, err := json.Marshal(result)
	if err != nil {
		return err
	}

	kv := api.KVPair{Key: result.Key(), Value: d}
	_, err = util.Consul().KV().Put(&kv, &api.WriteOptions{})
	return err
}
