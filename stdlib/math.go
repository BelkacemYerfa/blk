package stdlib

import (
	"blk/object"
	"math"
)

// math module definition
var mathModule = object.Module{
	// math constants
	// copied from
	// https://github.com/d5/tengo/blob/master/stdlib/math.go
	"e":                      &object.Float{Value: math.E},
	"pi":                     &object.Float{Value: math.Pi},
	"phi":                    &object.Float{Value: math.Phi},
	"sqrt2":                  &object.Float{Value: math.Sqrt2},
	"sqrtE":                  &object.Float{Value: math.SqrtE},
	"sqrtPi":                 &object.Float{Value: math.SqrtPi},
	"sqrtPhi":                &object.Float{Value: math.SqrtPhi},
	"ln2":                    &object.Float{Value: math.Ln2},
	"log2E":                  &object.Float{Value: math.Log2E},
	"ln10":                   &object.Float{Value: math.Ln10},
	"log10E":                 &object.Float{Value: math.Log10E},
	"maxFloat32":             &object.Float{Value: math.MaxFloat32},
	"smallestNonzeroFloat32": &object.Float{Value: math.SmallestNonzeroFloat32},
	"maxFloat64":             &object.Float{Value: math.MaxFloat64},
	"smallestNonzeroFloat64": &object.Float{Value: math.SmallestNonzeroFloat64},
	"maxInt":                 &object.Integer{Value: math.MaxInt},
	"minInt":                 &object.Integer{Value: math.MinInt},
	"maxInt8":                &object.Integer{Value: math.MaxInt8},
	"minInt8":                &object.Integer{Value: math.MinInt8},
	"maxInt16":               &object.Integer{Value: math.MaxInt16},
	"minInt16":               &object.Integer{Value: math.MinInt16},
	"maxInt32":               &object.Integer{Value: math.MaxInt32},
	"minInt32":               &object.Integer{Value: math.MinInt32},
	"maxInt64":               &object.Integer{Value: math.MaxInt64},
	"minInt64":               &object.Integer{Value: math.MinInt64},
	// math functions
	"abs":         &object.BuiltinFn{Fn: funcF64F64(math.Abs)},
	"acos":        &object.BuiltinFn{Fn: funcF64F64(math.Acos)},
	"acosh":       &object.BuiltinFn{Fn: funcF64F64(math.Acosh)},
	"asin":        &object.BuiltinFn{Fn: funcF64F64(math.Asin)},
	"asinh":       &object.BuiltinFn{Fn: funcF64F64(math.Asinh)},
	"atan":        &object.BuiltinFn{Fn: funcF64F64(math.Atan)},
	"atanh":       &object.BuiltinFn{Fn: funcF64F64(math.Atanh)},
	"cbrt":        &object.BuiltinFn{Fn: funcF64F64(math.Cbrt)},
	"ceil":        &object.BuiltinFn{Fn: funcF64F64(math.Ceil)},
	"floor":       &object.BuiltinFn{Fn: funcF64F64(math.Floor)},
	"cos":         &object.BuiltinFn{Fn: funcF64F64(math.Cos)},
	"cosh":        &object.BuiltinFn{Fn: funcF64F64(math.Cosh)},
	"gamma":       &object.BuiltinFn{Fn: funcF64F64(math.Gamma)},
	"log":         &object.BuiltinFn{Fn: funcF64F64(math.Log)},
	"log10":       &object.BuiltinFn{Fn: funcF64F64(math.Log10)},
	"log1p":       &object.BuiltinFn{Fn: funcF64F64(math.Log1p)},
	"log2":        &object.BuiltinFn{Fn: funcF64F64(math.Log2)},
	"logb":        &object.BuiltinFn{Fn: funcF64F64(math.Logb)},
	"j0":          &object.BuiltinFn{Fn: funcF64F64(math.J0)},
	"j1":          &object.BuiltinFn{Fn: funcF64F64(math.J1)},
	"erf":         &object.BuiltinFn{Fn: funcF64F64(math.Erf)},
	"erfc":        &object.BuiltinFn{Fn: funcF64F64(math.Erfc)},
	"erfcinv":     &object.BuiltinFn{Fn: funcF64F64(math.Erfcinv)},
	"erfinv":      &object.BuiltinFn{Fn: funcF64F64(math.Erfinv)},
	"Exp":         &object.BuiltinFn{Fn: funcF64F64(math.Exp)},
	"Exp2":        &object.BuiltinFn{Fn: funcF64F64(math.Exp2)},
	"Expm1":       &object.BuiltinFn{Fn: funcF64F64(math.Expm1)},
	"round":       &object.BuiltinFn{Fn: funcF64F64(math.Round)},
	"roundToEven": &object.BuiltinFn{Fn: funcF64F64(math.RoundToEven)},
	"sin":         &object.BuiltinFn{Fn: funcF64F64(math.Sin)},
	"sinh":        &object.BuiltinFn{Fn: funcF64F64(math.Sinh)},
	"sqrt":        &object.BuiltinFn{Fn: funcF64F64(math.Sqrt)},
	"tan":         &object.BuiltinFn{Fn: funcF64F64(math.Tan)},
	"tanh":        &object.BuiltinFn{Fn: funcF64F64(math.Tanh)},
	"Trunc":       &object.BuiltinFn{Fn: funcF64F64(math.Trunc)},
	"y0":          &object.BuiltinFn{Fn: funcF64F64(math.Y0)},
	"y1":          &object.BuiltinFn{Fn: funcF64F64(math.Y1)},
	"dim":         &object.BuiltinFn{Fn: func2F64F64(math.Dim)},
	"max":         &object.BuiltinFn{Fn: func2F64F64(math.Max)},
	"min":         &object.BuiltinFn{Fn: func2F64F64(math.Min)},
	"mod":         &object.BuiltinFn{Fn: func2F64F64(math.Mod)},
	"pow":         &object.BuiltinFn{Fn: func2F64F64(math.Pow)},
	"remainder":   &object.BuiltinFn{Fn: func2F64F64(math.Remainder)},
	"atan2":       &object.BuiltinFn{Fn: func2F64F64(math.Atan2)},
	"copysign":    &object.BuiltinFn{Fn: func2F64F64(math.Copysign)},
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
		if arg.Value < 0 {
			return newError("pow function doesn't accept values < 0")
		}
		return &object.Float{Value: math.Pow(float64(arg.Value), pow)}
	case *object.Float:
		if arg.Value < 0 {
			return newError("pow function doesn't accept values < 0")
		}
		return &object.Float{Value: math.Pow(arg.Value, pow)}

	default:
		return newError("wrong type, expected an integer or float, got %s",
			args[0].Type())
	}
}
