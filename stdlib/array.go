package stdlib

import (
	"blk/object"
)

// math module definition
var arrayModule = object.Module{
	"equals": &object.BuiltinFn{Fn: equals},
	"index":  &object.BuiltinFn{Fn: index},
}

func equals(args ...object.Object) object.Object {
	if len(args) != 2 {
		return newError("wrong number of arguments. got=%d, want=2",
			len(args))
	}

	// both args need to be a hashMap
	arr1, arr2 := object.Array{}, object.Array{}

	mapper1, _ := object.Cast(args[0])

	switch array := mapper1.(type) {
	case *object.Array:
		// do something
		arr1 = *array
	default:
		return newError("first arg needs to be a map in equals function")
	}

	mapper2, _ := object.Cast(args[1])

	switch array := mapper2.(type) {
	case *object.Array:
		// do something
		arr2 = *array
	default:
		return newError("second arg needs to be a map in equals function")
	}

	if len(arr1.Elements) != len(arr2.Elements) {
		return &object.Boolean{
			Value: false,
		}
	}

	for idx, value := range arr1.Elements {
		elem := arr2.Elements[idx]

		if elem == nil {
			return &object.Boolean{
				Value: false,
			}
		}

		// no need to compare the keys, accessing a value already means the keys are equals
		if !object.ObjectEquals(elem, value) {
			return &object.Boolean{
				Value: false,
			}
		}
	}

	return &object.Boolean{
		Value: true,
	}
}

func index(args ...object.Object) object.Object {
	if len(args) != 2 {
		return newError("wrong number of arguments. got=%d, want=2",
			len(args))
	}

	mapper, _ := object.Cast(args[0])

	actualArray := &object.Array{}
	switch array := mapper.(type) {
	case *object.Array:
		// do something
		actualArray = array
	default:
		return newError("second arg needs to be a map in equals function")
	}

	targetValue, _ := object.Cast(args[1])

	for idx, elem := range actualArray.Elements {
		if object.ObjectEquals(targetValue, elem) {
			return &object.Integer{
				Value: int64(idx),
			}
		}
	}

	return &object.Integer{
		Value: -1,
	}
}
