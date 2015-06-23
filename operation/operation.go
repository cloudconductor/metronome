package operation

type Operation interface {
	String() string
	SetPattern(pattern string)
	Run(vars map[string]string) error
}

type BaseOperation struct {
	pattern string
}

func (o *BaseOperation) SetPattern(pattern string) {
	o.pattern = pattern
}
