package internals

// This file handles an error collector obj

type ErrorCollector struct {
	Errors []error
}

func NewErrorCollector() *ErrorCollector {
	return &ErrorCollector{
		Errors: make([]error, 0),
	}
}

func (ec *ErrorCollector) Add(err error) {
	ec.Errors = append(ec.Errors, err)
}
