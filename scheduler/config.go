package scheduler

type Config struct {
	Variables map[string]string
	Default   TaskDefault
}

type TaskDefault struct {
	Timeout    int32
	ChefConfig string `json:"chef_config"`
}
