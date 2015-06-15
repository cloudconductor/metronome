package operation

type Operation interface {
	Name() string
	Run() error
}

type OperationFactory func(interface{}) (Operation, error)
