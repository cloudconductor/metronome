package task

type Task struct {
	Name        string
	Trigger     string
	Description string
	Filter      Filter
}

type Filter struct {
	Service string
	Tag     string
}
