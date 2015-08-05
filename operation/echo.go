package operation

import (
	"encoding/json"
	"metronome/util"

	log "github.com/Sirupsen/logrus"
)

type EchoOperation struct {
	BaseOperation
	message string
}

func NewEchoOperation(v json.RawMessage) *EchoOperation {
	o := &EchoOperation{}
	json.Unmarshal(v, &o.message)
	return o
}

func (o *EchoOperation) Run(vars map[string]string) error {
	log.Info("echo: " + util.ParseString(o.message, vars))
	return nil
}

func (o *EchoOperation) String() string {
	return "echo"
}
