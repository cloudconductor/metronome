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
		"echo": func(v json.RawMessage) (Operation, error) {
			var s string
			json.Unmarshal(v, &s)
			return &EchoOperation{message: s}, nil
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
