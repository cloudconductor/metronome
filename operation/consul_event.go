package operation

import (
	"encoding/json"
	"scheduler/util"

	log "github.com/Sirupsen/logrus"
	"github.com/hashicorp/consul/api"
)

type ConsulEventOperation struct {
	BaseOperation
	Name   string
	Filter struct {
		Service string
		Tag     string
	}
}

func NewConsulEventOperation(v json.RawMessage) *ConsulEventOperation {
	o := &ConsulEventOperation{}
	json.Unmarshal(v, &o)
	return o
}

func (o *ConsulEventOperation) Run(vars map[string]string) error {
	event := &api.UserEvent{
		Name:          o.Name,
		ServiceFilter: o.Filter.Service,
		TagFilter:     o.Filter.Tag,
	}
	id, _, err := util.Consul().Event().Fire(event, &api.WriteOptions{})
	log.Infof("consul-event: Fire %s event(ID: %s)", o.Name, id)
	return err
}

func (o *ConsulEventOperation) String() string {
	return "consul-event"
}
