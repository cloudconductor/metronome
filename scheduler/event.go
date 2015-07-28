package scheduler

import "fmt"

type Event struct {
	Pattern      string
	Name         string
	Description  string
	Priority     int
	OrderedTasks []EventTask `json:"ordered_tasks"`
	Task         string
}

type Events []Event

func (e *Event) SetPattern(pattern string) {
	e.Pattern = pattern
}

func (e *Event) Run(scheduler *Scheduler) error {
	var tasks []EventTask
	if e.Task != "" {
		tasks = []EventTask{EventTask{Pattern: e.Pattern, Task: e.Task}}
	} else {
		tasks = e.OrderedTasks
	}

	for _, et := range tasks {
		if err := et.Run(scheduler); err != nil {
			return err
		}
	}
	return nil
}

func (e Event) String() string {
	s := ""
	s += fmt.Sprintf("Name: %s\n", e.Name)
	s += fmt.Sprintf("Pattern: %s\n", e.Pattern)
	s += fmt.Sprintf("Description: %s\n", e.Description)
	s += fmt.Sprintf("Priority: %d\n", e.Priority)
	if e.Task != "" {
		s += fmt.Sprintf("Task: %s\n", e.Task)
	}

	if len(e.OrderedTasks) > 0 {
		s += "OrderedTasks:\n"
		for i, et := range e.OrderedTasks {
			if et.Tag == "" {
				s += fmt.Sprintf("  %d: Service: %s, Task: %s\n", i, et.Service, et.Task)
			} else {
				s += fmt.Sprintf("  %d: Service: %s, Tag: %s, Task: %s\n", i, et.Service, et.Tag, et.Task)
			}
		}
	}
	return s
}

func (e Events) Len() int {
	return len(e)
}

func (e Events) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

func (e Events) Less(i, j int) bool {
	return e[i].Priority < e[j].Priority
}
