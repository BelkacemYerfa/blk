package stdlib

import (
	"blk/object"
)

// type module definition
var typeModule = object.Module{
	"typeOf": &object.BuiltinFn{Fn: typeOf},
	"cast":   &object.BuiltinFn{Fn: cast},
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

func cast(args ...object.Object) object.Object {
	if len(args) != 2 {
		return newError("wrong number of arguments. got=%d, want=1",
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
		return newError("currently cast function support only operation on ints and floats")
	}

	return nil
}
