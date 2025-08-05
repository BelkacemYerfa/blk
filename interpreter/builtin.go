package interpreter

import (
	"blk/object"
)

// this offers built in function so u don't need module imports to use them
var builtInFunction = object.Module{
	"len": &object.BuiltinFn{Fn: LEN},
}

func LEN(args ...object.Object) object.Object {
	if len(args) != 1 {
		return newError("wrong number of arguments. got=%d, want=1",
			len(args))
	}
	switch arg := args[0].(type) {
	case *object.String:
		return &object.Integer{Value: int64(len(arg.Value))}
	default:
		return newError("argument to `len` not supported, got %s",
			args[0].Type())
	}
}

var builtInConstants = map[string]*object.BuiltinConst{}
