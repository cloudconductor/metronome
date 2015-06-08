package operation

var Operations map[string]OperationFactory

func init() {
	Operations = map[string]OperationFactory{
		"echo": func(v interface{}) (Operation, error) {
			return &EchoOperation{}, nil
		},
	}
}
