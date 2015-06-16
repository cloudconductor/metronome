package operation

type Operation interface {
	String() string
	Run() error
}
