package operation

import (
	"encoding/json"
	"metronome/config"
	"metronome/util"

	log "github.com/Sirupsen/logrus"
	"github.com/hashicorp/consul/api"
)

//	Trigger consul event with specified name
type ConsulEventOperation struct {
	BaseOperation
	Name   string
	Filter struct {
		Service string
		Tag     string
	}
}

func (o *ConsulEventOperation) SetDefault(m map[string]interface{}) {
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
		Payload:       []byte(config.Token),
	}

	id, _, err := util.Consul().Event().Fire(event, &api.WriteOptions{})
	log.Infof("consul-event: Fire %s event(ID: %s)", o.Name, id)
	return err
}

func (o *ConsulEventOperation) String() string {
	return "consul-event"
}
