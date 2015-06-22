package operation

type Operation interface {
	String() string
	Run(vars map[string]string) error
}
