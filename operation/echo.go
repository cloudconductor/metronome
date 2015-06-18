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

func (o *EchoOperation) Run(m map[string]string) error {
	fmt.Println(util.ParseString(o.message, m))
	return nil
}

func (o *EchoOperation) String() string {
	return "echo"
}
