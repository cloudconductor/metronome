package scheduler

import (
	"encoding/json"
	"fmt"
	"metronome/util"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/consul/api"
)

const EVENT_RESULT_KEY = "metronome/results"

type Result interface {
	Key() string
}

//	Result of entire event
type EventResult struct {
	ID         string
	Name       string
	Status     string
	StartedAt  time.Time
	FinishedAt time.Time
}

//	Result of task
type TaskResult struct {
	EventID    string
	No         int
	Name       string
	Status     string
	StartedAt  time.Time
	FinishedAt time.Time
}

//	Result of task on individual node
type NodeTaskResult struct {
	EventID    string
	No         int
	Node       string
	Status     string
	Log        string `json:"-"`
	StartedAt  time.Time
	FinishedAt time.Time
}

func (r *EventResult) MarshalJSON() ([]byte, error) {
	//	Marshal result that is excepted empty attributes to JSON format
	var fields []string
	fields = append(fields, fmt.Sprintf("\"ID\": \"%s\"", r.ID))
	fields = append(fields, fmt.Sprintf("\"Name\": \"%s\"", r.Name))
	fields = append(fields, fmt.Sprintf("\"Status\": \"%s\"", r.Status))
	if !r.StartedAt.IsZero() {
		fields = append(fields, fmt.Sprintf("\"StartedAt\": \"%s\"", r.StartedAt.Format(time.RFC3339)))
	}
	if !r.FinishedAt.IsZero() {
		fields = append(fields, fmt.Sprintf("\"FinishedAt\": \"%s\"", r.FinishedAt.Format(time.RFC3339)))
	}
	return []byte(fmt.Sprintf("{ %s }", strings.Join(fields, ","))), nil
}

func (r *TaskResult) MarshalJSON() ([]byte, error) {
	//	Marshal result that is excepted empty attributes to JSON format
	var fields []string
	fields = append(fields, fmt.Sprintf("\"EventID\": \"%s\"", r.EventID))
	fields = append(fields, fmt.Sprintf("\"No\": %d", r.No))
	fields = append(fields, fmt.Sprintf("\"Name\": \"%s\"", r.Name))
	fields = append(fields, fmt.Sprintf("\"Status\": \"%s\"", r.Status))
	if !r.StartedAt.IsZero() {
		fields = append(fields, fmt.Sprintf("\"StartedAt\": \"%s\"", r.StartedAt.Format(time.RFC3339)))
	}
	if !r.FinishedAt.IsZero() {
		fields = append(fields, fmt.Sprintf("\"FinishedAt\": \"%s\"", r.FinishedAt.Format(time.RFC3339)))
	}
	return []byte(fmt.Sprintf("{ %s }", strings.Join(fields, ","))), nil
}

func (r *NodeTaskResult) MarshalJSON() ([]byte, error) {
	//	Marshal result that is excepted empty attributes to JSON format
	var fields []string
	fields = append(fields, fmt.Sprintf("\"EventID\": \"%s\"", r.EventID))
	fields = append(fields, fmt.Sprintf("\"No\": %d", r.No))
	fields = append(fields, fmt.Sprintf("\"Node\": \"%s\"", r.Node))
	fields = append(fields, fmt.Sprintf("\"Status\": \"%s\"", r.Status))
	if !r.StartedAt.IsZero() {
		fields = append(fields, fmt.Sprintf("\"StartedAt\": \"%s\"", r.StartedAt.Format(time.RFC3339)))
	}
	if !r.FinishedAt.IsZero() {
		fields = append(fields, fmt.Sprintf("\"FinishedAt\": \"%s\"", r.FinishedAt.Format(time.RFC3339)))
	}
	return []byte(fmt.Sprintf("{ %s }", strings.Join(fields, ","))), nil
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
	//	Save any result to consul KVS
	if err := putResult(r); err != nil {
		return err
	}
	kv := &api.KVPair{
		Key:   r.Key() + "/log",
		Value: []byte(r.Log),
	}
	_, err := util.Consul().KV().Put(kv, &api.WriteOptions{})
	return err
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
	//	Collect all results on node that belongs with this task
	var results []NodeTaskResult

	prefix := EVENT_RESULT_KEY + "/" + r.EventID + "/" + strconv.Itoa(r.No)
	kvs, _, err := util.Consul().KV().List(prefix, &api.QueryOptions{})
	if err != nil {
		return nil, err
	}

	for _, kv := range kvs {
		//	Except log record on each node
		node := strings.TrimPrefix(kv.Key, prefix)
		if node == "" || strings.HasSuffix(node, "/log") {
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

	//	Read log from /metronome/result/[EventID]/[No]/[Node]/log
	kv, _, err := util.Consul().KV().Get(key+"/log", &api.QueryOptions{})
	if err != nil {
		return nil, err
	}
	if kv != nil {
		result.Log = string(kv.Value)
	}
	return &result, err
}

//	Get any result from consul KVS
func getResult(key string, result interface{}) (bool, error) {
	kv, _, err := util.Consul().KV().Get(key, &api.QueryOptions{})
	if err != nil {
		return false, err
	}

	if kv == nil || len(kv.Value) == 0 {
		return false, nil
	}

	if err := json.Unmarshal(kv.Value, &result); err != nil {
		return false, err
	}
	return true, nil
}

//	Put any result to consul KVS with JSON format
func putResult(result Result) error {
	d, err := json.Marshal(result)
	if err != nil {
		return err
	}

	kv := api.KVPair{
		Key:   result.Key(),
		Value: d,
	}
	_, err = util.Consul().KV().Put(&kv, &api.WriteOptions{})
	return err
}
