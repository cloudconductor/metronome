package operation

type Operation interface {
	String() string
	Run(m map[string]string) error
}
