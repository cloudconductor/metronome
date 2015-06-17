package operation

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type ExecuteOperation struct {
	Script string
}

func NewExecuteOperation(v json.RawMessage) *ExecuteOperation {
	o := &ExecuteOperation{}
	json.Unmarshal(v, &o.Script)
	return o
}

func (o *ExecuteOperation) Run(m map[string]string) error {
	cmd := exec.Command(os.Getenv("SHELL"))
	cmd.Stdin = strings.NewReader(o.Script)
	out, err := cmd.CombinedOutput()
	fmt.Println(string(out))
	return err
}

func (o *ExecuteOperation) String() string {
	return "execute"
}
