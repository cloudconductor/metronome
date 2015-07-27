package operation

import (
	"encoding/json"
	"errors"
	"fmt"
)

type OperationFactory func(json.RawMessage) (Operation, error)

var Operations map[string]OperationFactory

func init() {
	Operations = map[string]OperationFactory{
		"chef": func(v json.RawMessage) (Operation, error) {
			return NewChefOperation(v), nil
		},
		"echo": func(v json.RawMessage) (Operation, error) {
			return NewEchoOperation(v), nil
		},
		"service": func(v json.RawMessage) (Operation, error) {
			return NewServiceOperation(v), nil
		},
		"execute": func(v json.RawMessage) (Operation, error) {
			return NewExecuteOperation(v), nil
		},
		"task": func(v json.RawMessage) (Operation, error) {
			return NewTaskOperation(v), nil
		},
		"consul-kvs": func(v json.RawMessage) (Operation, error) {
			return NewConsulKVSOperation(v), nil
		},
		"consul-event": func(v json.RawMessage) (Operation, error) {
			return NewConsulEventOperation(v), nil
		},
	}
}

func UnmarshalOperations(d []byte, operations *[]Operation) error {
	var result []Operation
	var list []map[string]json.RawMessage
	err := json.Unmarshal(d, &list)
	if err != nil {
		return err
	}

	for _, m := range list {
		if len(m) != 1 {
			return errors.New("Operation has multiple types")
		}

		for k, v := range m {
			factory, ok := Operations[k]
			if !ok {
				return errors.New(fmt.Sprintf("Operation %s is not defined", k))
			}
			o, err := factory(v)
			if err != nil {
				return err
			}
			result = append(result, o)
		}
	}
	*operations = result
	return nil
}
