package operation

import (
	"encoding/json"
	"errors"
	"fmt"
	"scheduler/util"

	"github.com/hashicorp/consul/api"
)

type ConsulKVSOperation struct {
	BaseOperation
	Action string
	Key    string
	Value  string
}

func NewConsulKVSOperation(v json.RawMessage) *ConsulKVSOperation {
	o := &ConsulKVSOperation{}
	json.Unmarshal(v, &o)
	return o
}

func (o *ConsulKVSOperation) Run(vars map[string]string) error {
	switch o.Action {
	case "put":
		return o.put(vars)
	case "delete":
		return o.delete(vars)
	default:
		return errors.New(fmt.Sprintf("[consul-kvs]Operation can't support %s action", o.Action))
	}
	return nil
}

func (o *ConsulKVSOperation) put(vars map[string]string) error {
	kv := &api.KVPair{Key: o.Key, Value: []byte(o.Value)}
	_, err := util.Consul().KV().Put(kv, &api.WriteOptions{})
	fmt.Printf("[consul-kvs]: Put %s to %s.\n", o.Value, o.Key)
	return err
}

func (o *ConsulKVSOperation) delete(vars map[string]string) error {
	_, err := util.Consul().KV().Delete(o.Key, &api.WriteOptions{})
	fmt.Printf("[consul-kvs]: Delete %s.\n", o.Key)
	return err
}

func (o *ConsulKVSOperation) String() string {
	return "consul-kvs"
}
