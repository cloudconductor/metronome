package operation

import (
	"encoding/json"
	"metronome/util"
	"os/exec"
	"path/filepath"
	"strings"

	log "github.com/Sirupsen/logrus"
)

//	Execute specified script or executable file
type ExecuteOperation struct {
	BaseOperation
	File      string
	Arguments []string
	Script    string
	Output    bool
}

func NewExecuteOperation(v json.RawMessage) *ExecuteOperation {
	o := &ExecuteOperation{}
	o.Output = true
	json.Unmarshal(v, &o)
	return o
}

func (o *ExecuteOperation) Run(vars map[string]string) error {
	cmd := &exec.Cmd{}
	cmd.Dir = filepath.Dir(o.path)
	if o.File != "" {
		//	Execute target file with arguuments
		file := util.ParseString(o.File, vars)
		cmd.Path = file
		cmd.Args = append([]string{file}, o.Arguments...)
	} else {
		//	Execute script in the shell
		cmd.Path = "/bin/sh"
		s := util.ParseString(o.Script, vars)
		cmd.Stdin = strings.NewReader(s)
	}
	out, err := cmd.CombinedOutput()

	//	Output STDOUT if output flag in task.yml is true
	if o.Output {
		log.Info(string(out))
	}
	return err
}

func (o *ExecuteOperation) String() string {
	return "execute"
}
