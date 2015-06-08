package operation

type Operation interface {
	Run() error
}

type OperationFactory func(interface{}) (Operation, error)
