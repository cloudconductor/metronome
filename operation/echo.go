package operation

import (
	"fmt"
)

type EchoOperation struct {
}

func (t *EchoOperation) Run() error {
	fmt.Println("echo")
	return nil
}

func (t *EchoOperation) Name() string {
	return "echo"
}