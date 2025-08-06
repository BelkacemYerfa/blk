package stdlib

import (
	"blk/object"
)

// math module definition
var typeModule = object.Module{
	"typeOf": &object.BuiltinFn{Fn: typeOf},
}

func typeOf(args ...object.Object) object.Object {
	if len(args) != 1 {
		return newError("wrong number of arguments. got=%d, want=1",
			len(args))
	}

	return &object.String{
		Value: string(args[0].Type()),
	}
}
