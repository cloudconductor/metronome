package operation

import (
	"encoding/json"
	"fmt"
	"scheduler/util"
)

type EchoOperation struct {
	message string
}

func NewEchoOperation(v json.RawMessage) *EchoOperation {
	o := &EchoOperation{}
	json.Unmarshal(v, &o.message)
	return o
}

func (o *EchoOperation) Run(vars map[string]string) error {
	fmt.Println(util.ParseString(o.message, vars))
	return nil
}

func (o *EchoOperation) String() string {
	return "echo"
}
