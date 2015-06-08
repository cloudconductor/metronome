package scheduler

import "scheduler/task"

type Config struct {
	Variables map[string]string
	Default   TaskDefault
	Tasks     []task.Task
}

type TaskDefault struct {
	Timeout    int32
	ChefConfig string `json:"chef_config"`
}
