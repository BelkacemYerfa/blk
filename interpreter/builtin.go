package interpreter

import (
	"blk/object"
	"fmt"
)

// this offers built in function so u don't need module imports to use them
var builtInFunction = object.Module{
	"len":   &object.BuiltinFn{Fn: LEN},
	"print": &object.BuiltinFn{Fn: PRINT},
}

func LEN(args ...object.Object) object.Object {
	if len(args) != 1 {
		return newError("wrong number of arguments. got=%d, want=1",
			len(args))
	}

	arg, _ := object.Cast(args[0])

	switch arg := arg.(type) {
	case *object.String:
		return &object.Integer{Value: int64(len(arg.Value))}
	case *object.Array:
		return &object.Integer{Value: int64(len(arg.Elements))}
	default:
		return newError("argument to `len` not supported, got %s",
			args[0].Type())
	}
}

// this is a print function for test only
func PRINT(args ...object.Object) object.Object {
	results := []object.Object{}
	for _, arg := range args {
		arg, _ = object.Cast(arg)
		results = append(results, arg)
		fmt.Print(arg.Inspect())
		if len(args) > 1 {
			fmt.Print(" ")
		}
	}
	fmt.Println()
	return &object.Array{
		Elements: results,
	}
}

var builtInConstants = map[string]*object.BuiltinConst{}
