package interpreter

import (
	"blk/object"
	"fmt"
	"strings"
)

// this offers built in function so u don't need module imports to use them
var builtInFunction = object.Module{
	"len":   &object.BuiltinFn{Fn: LEN},
	"copy":  &object.BuiltinFn{Fn: COPY},
	"print": &object.BuiltinFn{Fn: PRINT},
}

func LEN(args ...object.Object) object.Object {
	if len(args) != 1 {
		return newError(ERROR, "wrong number of arguments. got=%d, want=1",
			len(args))
	}

	arg, _ := object.Cast(args[0])

	switch arg := arg.(type) {
	case *object.String:
		return &object.Integer{Value: int64(len(arg.Value))}
	case *object.Array:
		return &object.Integer{Value: int64(len(arg.Elements))}
	default:
		return newError(ERROR, "argument to `len` not supported, got %s",
			args[0].Type())
	}
}

// function works on creating deep copies on gives params
func COPY(args ...object.Object) object.Object {
	if len(args) != 1 {
		return newError(ERROR, "wrong number of arguments. got=%d, want=1",
			len(args))
	}

	arg, _ := object.Cast(args[0])

	return object.DeepCopy(arg)
}

// this is a print function for test only
func PRINT(args ...object.Object) object.Object {
	results := []string{}
	for _, arg := range args {
		arg, _ = object.Cast(arg)
		results = append(results, arg.Inspect())
	}
	fmt.Println(strings.Join(results, " "))
	return nil
}

var builtInConstants = map[string]*object.BuiltinConst{}
