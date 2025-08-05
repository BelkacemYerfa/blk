package stdlib

import (
	"blk/object"
	"fmt"
)

func newError(format string, a ...interface{}) *object.Error {
	return &object.Error{Message: fmt.Sprintf(format, a...)}
}

// every module added to the std lib needs to be defined here with a name
var BuiltinModules = map[string]object.Module{
	"math": mathModule,
}
