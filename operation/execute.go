package operation

import (
	"encoding/json"
	"metronome/config"
	"metronome/util"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	log "github.com/Sirupsen/logrus"
)

type ExecuteOperation struct {
	BaseOperation
	File   string
	Script string
	Output bool
}

func NewExecuteOperation(v json.RawMessage) *ExecuteOperation {
	o := &ExecuteOperation{}
	o.Output = true
	json.Unmarshal(v, &o)
	return o
}

func (o *ExecuteOperation) Run(vars map[string]string) error {
	cmd := exec.Command(config.Shell)
	cmd.Dir = filepath.Dir(o.path)
	if o.File != "" {
		file := util.ParseString(o.File, vars)
		cmd.Args = append(cmd.Args, file)
	} else {
		s := util.ParseString(o.Script, vars)
		cmd.Stdin = strings.NewReader(s)
	}
	out, err := cmd.CombinedOutput()

	if o.Output {
		log.Info(string(out))
	}
	return err
}

func (o *ExecuteOperation) String() string {
	return "execute"
}
