package stdlib

import (
	"blk/object"
	"math"
)

// math module definition
var mathModule = object.Module{
	// math constants
	// TODO: consider adding more
	"PI": &object.BuiltinConst{Const: &object.Float{Value: math.Pi}},
	"e":  &object.BuiltinConst{Const: &object.Float{Value: math.E}},
	// math functions
	// TODO: consider adding more
	"abs": &object.BuiltinFn{Fn: ABS},
	"pow": &object.BuiltinFn{Fn: POW},
}

func ABS(args ...object.Object) object.Object {
	if len(args) != 1 {
		return newError("wrong number of arguments. got=%d, want=1",
			len(args))
	}

	num, _ := object.Cast(args[0])

	switch arg := num.(type) {
	case *object.Integer:
		return &object.Integer{Value: int64(math.Abs(float64(arg.Value)))}
	case *object.Float:
		return &object.Float{Value: math.Abs(float64(arg.Value))}

	default:
		return newError("wrong type, expected an integer or float, got %s",
			args[0].Type())
	}
}

func POW(args ...object.Object) object.Object {
	if len(args) != 2 {
		return newError("wrong number of arguments. got=%d, want=2",
			len(args))
	}

	power, _ := object.Cast(args[1])
	num, _ := object.Cast(args[0])

	// values either they're int or float
	pow := float64(1)
	switch p := power.(type) {
	case *object.Integer:
		pow = float64(p.Value)
	case *object.Float:
		pow = p.Value
	default:
		return newError("wrong type, expected the pow to be an integer or float, got %s", args[1].Type())
	}

	switch arg := num.(type) {
	case *object.Integer:
		return &object.Float{Value: math.Pow(float64(arg.Value), pow)}
	case *object.Float:
		return &object.Float{Value: math.Pow(arg.Value, pow)}

	default:
		return newError("wrong type, expected an integer or float, got %s",
			args[0].Type())
	}
}
