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

func (t *EchoOperation) Run() error {
	fmt.Println("-----------echo")
	fmt.Println(t.message)
	return nil
}

func (t *EchoOperation) Name() string {
	return "echo"
}
