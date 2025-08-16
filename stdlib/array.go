package stdlib

import (
	"blk/object"
)

// math module definition
var arrayModule = object.Module{
	"equals": &object.BuiltinFn{Fn: arrayEquals},
	"index":  &object.BuiltinFn{Fn: index},
	"append": &object.BuiltinFn{Fn: APPEND},
}

func arrayEquals(args ...object.Object) object.Object {
	if len(args) != 2 {
		return newError("wrong number of arguments. got=%d, want=3",
			len(args))
	}

	if args[0].Type() != object.ARRAY_OBJ && args[1].Type() != object.ARRAY_OBJ {
		return newError("both args need to be an array in equals function")
	}

	return &object.Boolean{
		Value: args[0].Equals(args[1]),
	}
}

func index(args ...object.Object) object.Object {
	if len(args) != 2 {
		return newError("wrong number of arguments. got=%d, want=2",
			len(args))
	}

	mapper, _ := object.Cast(args[0])
	target, _ := object.Cast(args[1])

	switch actualArray := mapper.(type) {
	case *object.Array:
		// do something
		for idx, elem := range actualArray.Elements {
			if elem.Equals(target) {
				return &object.Integer{
					Value: int64(idx),
				}
			}
		}
		return &object.Integer{
			Value: -1,
		}
	default:
		return newError("second arg needs to be an array in equals function")
	}

}

func APPEND(args ...object.Object) object.Object {
	if len(args) != 2 {
		return newError("wrong number of arguments. got=%d, want=2",
			len(args))
	}

	array, isMutable := object.Cast(args[0])

	if !isMutable {
		return newError("can't mutate %v, probably defined as a const", args[0].Inspect())
	}

	switch actualArray := array.(type) {
	case *object.Array:
		// cast value
		newValue, _ := object.Cast(args[1])

		// means that the array reached it limits
		if actualArray.Size == len(actualArray.Elements) {
			return newError("can't append more value to this array, since it reached the max len allowed for it, initialization %d, current %d", actualArray.Size, len(actualArray.Elements))
		}

		// type checks if the new value to insert has corresponding type to the current ones on the array
		if len(actualArray.Elements) > 0 {
			elem := actualArray.Elements[0]
			if elem.Type() != newValue.Type() {
				return newError("can't append the current value, cause it doesn't match the type of the current elements which is of type %s", elem.Type())
			}
		}

		actualArray.Elements = append(actualArray.Elements, newValue)
		return actualArray

	default:
		return newError("first args needs to be an array")
	}

}
