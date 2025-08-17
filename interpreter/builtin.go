package interpreter

import (
	"blk/object"
	"fmt"
	"strings"
)

// this offers built in function so u don't need module imports to use them
var builtInFunction = object.Module{
	"len":  &object.BuiltinFn{Fn: LEN},
	"copy": &object.BuiltinFn{Fn: COPY},
	"cast": &object.BuiltinFn{Fn: cast},
	"INTEGER": &object.BuiltinConst{
		Const: &object.String{
			Value: "INTEGER",
		},
	},
	"FLOAT": &object.BuiltinConst{
		Const: &object.String{
			Value: "FLOAT",
		},
	},
	"print": &object.BuiltinFn{Fn: PRINT},
}

func cast(args ...object.Object) object.Object {
	if len(args) != 2 {
		return newError(ERROR, "wrong number of arguments. got=%d, want=1",
			len(args))
	}

	// first arg is the value
	// second arg is the type to cast into

	castedVal, _ := object.Cast(args[0])
	castType, _ := object.Cast(args[1])

	switch v := castedVal.(type) {
	// TODO: change the way of handling the type inspection
	case *object.Integer:
		if castType.Inspect() == object.INTEGER_OBJ {
			return &object.Integer{
				Value: v.Value,
			}
		}

		if castType.Inspect() == object.FLOAT_OBJ {
			return &object.Float{
				Value: float64(v.Value),
			}
		}
	case *object.Float:
		if castType.Inspect() == object.INTEGER_OBJ {
			return &object.Integer{
				Value: int64(v.Value),
			}
		}

		if castType.Inspect() == object.FLOAT_OBJ {
			return &object.Float{
				Value: v.Value,
			}
		}
	default:
		return newError(ERROR, "currently cast function support only operation on ints and floats")
	}

	return nil
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

	return args[0].Copy()
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
