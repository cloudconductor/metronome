package operation

import (
	"fmt"
)

type EchoOperation struct {
	message string
}

func (t *EchoOperation) Run() error {
	fmt.Println("-----------echo")
	fmt.Println(t.message)
	return nil
}

func (t *EchoOperation) Name() string {
	return "echo"
}
