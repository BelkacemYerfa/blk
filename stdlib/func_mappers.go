package stdlib

import (
	"blk/object"
)

// maps a function type fn func(string, string) bool
// to a builtin function
func funcSSB(fn func(string, string) bool) object.BuiltinFunction {
	return func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return newError("wrong number of arguments. got=%d, want=2",
				len(args))
		}

		args[0], _ = object.Cast(args[0])
		args[1], _ = object.Cast(args[1])

		// check that firstArg, secondArg is string
		if args[0].Type() != object.STRING_OBJ || args[1].Type() != object.STRING_OBJ {
			return newError("both args need to be of type string")
		}

		firstArg := args[0].(*object.String)
		secondArg := args[1].(*object.String)

		return &object.Boolean{
			Value: fn(firstArg.Value, secondArg.Value),
		}
	}
}

// maps a function type fn func(string, string) int
// to a builtin function
func funcSSI(fn func(string, string) int) object.BuiltinFunction {
	return func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return newError("wrong number of arguments. got=%d, want=2",
				len(args))
		}

		args[0], _ = object.Cast(args[0])
		args[1], _ = object.Cast(args[1])

		// check that firstArg, secondArg is string
		if args[0].Type() != object.STRING_OBJ || args[1].Type() != object.STRING_OBJ {
			return newError("both args need to be of type string")
		}

		firstArg := args[0].(*object.String)
		secondArg := args[1].(*object.String)

		return &object.Integer{
			Value: int64(fn(firstArg.Value, secondArg.Value)),
		}
	}
}

// maps a function type fn func(string, string) string
// to a builtin function
func func2SS(fn func(string, string) string) object.BuiltinFunction {
	return func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return newError("wrong number of arguments. got=%d, want=2",
				len(args))
		}

		args[0], _ = object.Cast(args[0])
		args[1], _ = object.Cast(args[1])

		// check that firstArg, secondArg is string
		if args[0].Type() != object.STRING_OBJ || args[1].Type() != object.STRING_OBJ {
			return newError("both args need to be of type string")
		}

		firstArg := args[0].(*object.String)
		secondArg := args[1].(*object.String)

		return &object.String{
			Value: fn(firstArg.Value, secondArg.Value),
		}
	}
}

// maps a function type fn func(string) string
// to a builtin function
func funcSS(fn func(string) string) object.BuiltinFunction {
	return func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return newError("wrong number of arguments. got=%d, want=1",
				len(args))
		}

		args[0], _ = object.Cast(args[0])

		// check that firstArg, secondArg is string
		if args[0].Type() != object.STRING_OBJ {
			return newError("both args need to be of type string")
		}

		firstArg := args[0].(*object.String)

		return &object.String{
			Value: fn(firstArg.Value),
		}
	}
}

// maps a function type fn func(float64) float64
// to a builtin function
func funcF64F64(fn func(float64) float64) object.BuiltinFunction {
	return func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return newError("wrong number of arguments. got=%d, want=1",
				len(args))
		}

		args[0], _ = object.Cast(args[0])
		args[1], _ = object.Cast(args[1])

		// check that firstArg, secondArg is string
		if args[0].Type() != object.FLOAT_OBJ || args[1].Type() != object.FLOAT_OBJ {
			return newError("both args need to be of type float")
		}

		firstArg := args[0].(*object.Float)

		return &object.Float{
			Value: fn(firstArg.Value),
		}
	}
}

// maps a function type fn func(float64, float64) float64
// to a builtin function
func func2F64F64(fn func(float64, float64) float64) object.BuiltinFunction {
	return func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return newError("wrong number of arguments. got=%d, want=1",
				len(args))
		}

		args[0], _ = object.Cast(args[0])

		// check that firstArg, secondArg is string
		if args[0].Type() != object.FLOAT_OBJ {
			return newError("both args need to be of type float")
		}

		firstArg := args[0].(*object.Float)
		secondArg := args[1].(*object.Float)

		return &object.Float{
			Value: fn(firstArg.Value, secondArg.Value),
		}
	}
}
