package operation

import (
	"encoding/json"
	"os"
	"os/exec"
	"scheduler/util"
	"strings"

	log "github.com/Sirupsen/logrus"
)

type ExecuteOperation struct {
	BaseOperation
	Script string
}

func NewExecuteOperation(v json.RawMessage) *ExecuteOperation {
	o := &ExecuteOperation{}
	json.Unmarshal(v, &o.Script)
	return o
}

func (o *ExecuteOperation) Run(vars map[string]string) error {
	s := util.ParseString(o.Script, vars)

	cmd := exec.Command(os.Getenv("SHELL"))
	cmd.Stdin = strings.NewReader(s)
	out, err := cmd.CombinedOutput()
	log.Debug(string(out))
	return err
}

func (o *ExecuteOperation) String() string {
	return "execute"
}
