package operation

import (
	"encoding/json"
	"errors"
	"fmt"
	"metronome/util"

	log "github.com/Sirupsen/logrus"
	"github.com/hashicorp/consul/api"
)

//	Get, put, or delete on the consul KVS
type ConsulKVSOperation struct {
	BaseOperation
	Action string
	Key    string
	Value  string
	Name   string
}

func NewConsulKVSOperation(v json.RawMessage) *ConsulKVSOperation {
	o := &ConsulKVSOperation{}
	json.Unmarshal(v, &o)
	return o
}

func (o *ConsulKVSOperation) Run(vars map[string]string) error {
	switch o.Action {
	case "get":
		return o.get(vars)
	case "put":
		return o.put(vars)
	case "delete":
		return o.delete(vars)
	default:
		return errors.New(fmt.Sprintf("Operation can't support %s action", o.Action))
	}
	return nil
}

func (o *ConsulKVSOperation) get(vars map[string]string) error {
	//	Store value that has been get to variables map
	kv := &api.KVPair{
		Key:   o.Key,
		Value: []byte(o.Value),
	}
	kv, _, err := util.Consul().KV().Get(o.Key, &api.QueryOptions{})
	vars[o.Name] = string(kv.Value)
	log.Infof("Get %s from %s and store to %s", kv.Value, kv.Key, o.Name)
	return err
}

func (o *ConsulKVSOperation) put(vars map[string]string) error {
	kv := &api.KVPair{
		Key:   o.Key,
		Value: []byte(o.Value),
	}
	_, err := util.Consul().KV().Put(kv, &api.WriteOptions{})
	log.Infof("Put %s to %s", o.Value, o.Key)
	return err
}

func (o *ConsulKVSOperation) delete(vars map[string]string) error {
	_, err := util.Consul().KV().Delete(o.Key, &api.WriteOptions{})
	log.Infof("Delete %s", o.Key)
	return err
}

func (o *ConsulKVSOperation) String() string {
	return "consul-kvs"
}
