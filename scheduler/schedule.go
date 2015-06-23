package scheduler

import "scheduler/task"

type Schedule struct {
	pattern   string
	Variables map[string]string
	Default   TaskDefault
	Tasks     []task.Task
}

type TaskDefault struct {
	Timeout    int32
	ChefConfig string `json:"chef_config"`
}

func (s *Schedule) SetPattern(pattern string) {
	s.pattern = pattern
	for _, t := range s.Tasks {
		t.SetPattern(pattern)
	}
}
