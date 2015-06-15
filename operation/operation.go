package operation

type Operation interface {
	Name() string
	Run() error
}
