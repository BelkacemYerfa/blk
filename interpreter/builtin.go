package interpreter

import (
	"blk/object"
	"fmt"
	"strconv"
	"unicode/utf8"
)

// this offers built in function so u don't need module imports to use them
var builtInFunction = object.Module{
	"len":    &object.BuiltinFn{Fn: size},
	"copy":   &object.BuiltinFn{Fn: clone},
	"int":    &object.BuiltinFn{Fn: toInt},
	"float":  &object.BuiltinFn{Fn: toFloat},
	"string": &object.BuiltinFn{Fn: toString},
	"bool":   &object.BuiltinFn{Fn: toBool},
	"char":   &object.BuiltinFn{Fn: toChar},
	"typeOf": &object.BuiltinFn{Fn: typeOf},
}

func size(args ...object.Object) object.Object {
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
func clone(args ...object.Object) object.Object {
	if len(args) != 1 {
		return newError(ERROR, "wrong number of arguments. got=%d, want=1",
			len(args))
	}

	return args[0].Copy()
}

func toInt(args ...object.Object) object.Object {
	if len(args) != 1 {
		return newError(ERROR, "wrong number of arguments. got=%d, want=1",
			len(args))
	}

	// check the type of the args[0]

	if args[0].Type() != object.FLOAT_OBJ && args[0].Type() != object.STRING_OBJ {
		return newError(ERROR, "argument needs to be of type float or string, got %s", args[0].Type())
	}

	arg, _ := object.Cast(args[0])

	switch arg := arg.(type) {
	case *object.Float:
		return &object.Integer{
			Value: int64(arg.Value),
		}

	case *object.String:
		converted, err := strconv.Atoi(arg.Value)
		if err != nil {
			return newError(ERROR, err.Error())
		}
		return &object.Integer{
			Value: int64(converted),
		}
	}

	return nil
}

func toFloat(args ...object.Object) object.Object {
	if len(args) != 1 {
		return newError(ERROR, "wrong number of arguments. got=%d, want=1",
			len(args))
	}

	// check the type of the args[0]

	if args[0].Type() != object.INTEGER_OBJ && args[0].Type() != object.STRING_OBJ {
		return newError(ERROR, "argument needs to be of type int or string, got %s", args[0].Type())
	}

	arg, _ := object.Cast(args[0])

	switch arg := arg.(type) {
	case *object.Integer:
		return &object.Float{
			Value: float64(arg.Value),
		}

	case *object.String:
		converted, err := strconv.ParseFloat(arg.Value, 64)
		if err != nil {
			return newError(ERROR, err.Error())
		}
		return &object.Float{
			Value: (converted),
		}
	}

	return newError(ERROR, "unsupported input type %s", args[0].Type())
}

func toString(args ...object.Object) object.Object {
	if len(args) != 1 {
		return newError(ERROR, "wrong number of arguments. got=%d, want=1",
			len(args))
	}

	arg, _ := object.Cast(args[0])

	// check the type of the args[0]

	switch arg := arg.(type) {
	case *object.Integer:
		return &object.String{
			Value: fmt.Sprint(arg.Value),
		}

	case *object.Float:
		return &object.String{
			Value: fmt.Sprint(arg.Value),
		}

	case *object.Char:
		return &object.String{
			Value: string(arg.Value),
		}

	case *object.Boolean:
		return &object.String{
			Value: fmt.Sprint(arg.Value),
		}

	case *object.String:
		return arg
	}

	return newError(ERROR, "unsupported input type %s", args[0].Type())
}

func toBool(args ...object.Object) object.Object {
	if len(args) != 1 {
		return newError(ERROR, "wrong number of arguments. got=%d, want=1",
			len(args))
	}

	// check the type of the args[0]

	if args[0].Type() != object.STRING_OBJ {
		return newError(ERROR, "argument needs to be of type string, got %s", args[0].Type())
	}

	arg, _ := object.Cast(args[0])

	switch arg := arg.(type) {

	case *object.String:
		converted, err := strconv.ParseBool(arg.Value)
		if err != nil {
			return newError(ERROR, err.Error())
		}
		return &object.Boolean{
			Value: converted,
		}
	}

	return newError(ERROR, "unsupported input type %s", args[0].Type())
}

func toChar(args ...object.Object) object.Object {
	if len(args) != 1 {
		return newError(ERROR, "wrong number of arguments. got=%d, want=1",
			len(args))
	}

	// check the type of the args[0]

	if args[0].Type() != object.STRING_OBJ {
		return newError(ERROR, "argument needs to be of type string, got %s", args[0].Type())
	}

	arg, _ := object.Cast(args[0])

	switch arg := arg.(type) {

	case *object.String:
		if len(arg.Value) > 1 {
			return newError(ERROR, "converting a string to a rune requires string length=1, got=%d", len(arg.Value))
		}
		converted, _ := utf8.DecodeRuneInString(arg.Value)
		if converted == '\uFFFD' {
			return newError(ERROR, "string is empty")
		}
		return &object.Char{
			Value: converted,
		}
	}

	return newError(ERROR, "unsupported input type %s", args[0].Type())
}

func typeOf(args ...object.Object) object.Object {
	if len(args) != 1 {
		return newError(ERROR, "wrong number of arguments. got=%d, want=1",
			len(args))
	}

	return &object.String{
		Value: string(args[0].Type()),
	}
}

var builtInConstants = map[string]*object.BuiltinConst{}
