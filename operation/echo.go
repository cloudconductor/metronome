package operation

import (
	"encoding/json"
	"fmt"
)

type EchoOperation struct {
	message string
}

func NewEchoOperation(v json.RawMessage) *EchoOperation {
	o := &EchoOperation{}
	json.Unmarshal(v, &o.message)
	return o
}

func (o *EchoOperation) Run() error {
	fmt.Println("-----------echo")
	fmt.Println(o.message)
	return nil
}

func (o *EchoOperation) Name() string {
	return "echo"
}
