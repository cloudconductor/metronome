package operation

//	Extract common parameters and method from each operation
type Operation interface {
	String() string
	SetPattern(path string, pattern string)
	Run(vars map[string]string) error
}

type BaseOperation struct {
	path    string
	pattern string
}

func (o *BaseOperation) SetPattern(path string, pattern string) {
	o.path = path
	o.pattern = pattern
}
