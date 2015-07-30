package operation

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"scheduler/config"
	"scheduler/util"

	log "github.com/Sirupsen/logrus"
)

type ServiceOperation struct {
	BaseOperation
	Name   string
	Action string
}

func NewServiceOperation(v json.RawMessage) *ServiceOperation {
	o := &ServiceOperation{}
	json.Unmarshal(v, o)
	return o
}

func (o *ServiceOperation) Run(vars map[string]string) error {
	name := util.ParseString(o.Name, vars)
	action := util.ParseString(o.Action, vars)

	var cmd *exec.Cmd
	switch config.ServiceManager {
	case "init":
		cmd = exec.Command("/sbin/service", name, action)
	case "systemd":
		cmd = exec.Command("/sbin/systemctl", action, name)
	default:
		return errors.New(fmt.Sprintf("Unknown service manager(%s)", config.ServiceManager))
	}

	out, err := cmd.CombinedOutput()
	log.Debug(string(out))
	return err
}

func (o *ServiceOperation) String() string {
	return "service"
}
