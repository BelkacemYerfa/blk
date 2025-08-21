package stdlib

import (
	"blk/object"
)

var hashmapModule = object.Module{
	"keys":     &object.BuiltinFn{Fn: KEYS},
	"values":   &object.BuiltinFn{Fn: VALUES},
	"equals":   &object.BuiltinFn{Fn: EQUALS},
	"insert":   &object.BuiltinFn{Fn: INSERT},
	"getValue": &object.BuiltinFn{Fn: GET_VALUE},
	"delete":   &object.BuiltinFn{Fn: DELETE},
}

func KEYS(args ...object.Object) object.Object {
	if len(args) != 1 {
		return newError("wrong number of arguments. got=%d, want=1",
			len(args))
	}

	mapper, _ := object.Cast(args[0])

	// the type of args[0] needs to be a map
	switch hashMap := mapper.(type) {
	case *object.Map:
		// do something
		result := make([]object.Object, 0, len(hashMap.Pairs))

		for _, value := range hashMap.Pairs {
			result = append(result, value.Key)
		}

		return &object.Array{
			Elements: result,
		}
	default:
		return newError("args of keys functions needs to be a map")
	}
}

func VALUES(args ...object.Object) object.Object {
	if len(args) != 1 {
		return newError("wrong number of arguments. got=%d, want=1",
			len(args))
	}

	mapper, _ := object.Cast(args[0])

	// the type of args[0] needs to be a map
	switch hashMap := mapper.(type) {
	case *object.Map:
		// do something
		result := make([]object.Object, 0, len(hashMap.Pairs))

		for _, value := range hashMap.Pairs {
			result = append(result, value.Value)
		}

		return &object.Array{
			Elements: result,
		}
	default:
		return newError("args of keys functions needs to be a map")
	}
}

func EQUALS(args ...object.Object) object.Object {
	if len(args) != 2 {
		return newError("wrong number of arguments. got=%d, want=3",
			len(args))
	}

	if args[0].Type() != object.MAP_OBJ && args[1].Type() != object.MAP_OBJ {
		return newError("both args need to be a map in equals function")
	}

	return &object.Boolean{
		Value: args[0].Equals(args[1]),
	}
}

func INSERT(args ...object.Object) object.Object {
	if len(args) != 3 {
		return newError("wrong number of arguments. got=%d, want=3",
			len(args))
	}

	// first arg is map
	// second & third are the key value pair

	// check if the map can be mutated
	mapper, isMutable := object.Cast(args[0])

	if !isMutable {
		return newError("can't mutate %v, probably defined as a const", args[0].Inspect())
	}

	switch actualMap := mapper.(type) {
	case *object.Map:

		// cast the args
		newKey, _ := object.Cast(args[1])
		newValue, _ := object.Cast(args[2])

		// key needs to implement Hashable interface
		hashKey, ok := newKey.(object.Hashable)
		if !ok {
			return newError("unusable as hash key: %s, consider one of those types (boolean, integer, float, string)", newKey.Type())
		}

		// if map is empty push whatever there is as key-value
		if len(actualMap.Pairs) == 0 {
			actualMap.Pairs[hashKey.HashKey()] = object.HashPair{
				Key:   newKey,
				Value: newValue,
			}

			return actualMap
		}

		// if the map has at least one element

		// ? maybe it can be changed to get a random value from the hashMap without care to much about, didn't found a good way
		// check if the type of the value and the key are compatible with the ones in the hash map
		// need to do this only once
		for _, val := range actualMap.Pairs {
			key := val.Key
			value := val.Value

			if key.Type() != newKey.Type() {
				return newError("unusable as hash key: %s, doesn't match the current key(s) type(s): %s", newKey.Type(), key.Type())
			}

			if value.Type() != newValue.Type() {
				return newError("unusable as value key: %s, consider one of those types (boolean, integer, float, string)", newKey.Type())
			}

			actualMap.Pairs[hashKey.HashKey()] = object.HashPair{
				Key:   newKey,
				Value: newValue,
			}
			// break directly because it only needed once todo the check
			break
		}

		return mapper

	default:
		return newError("second arg needs to be a map in equals function")
	}
}

func GET_VALUE(args ...object.Object) object.Object {
	if len(args) != 2 {
		return newError("wrong number of arguments. got=%d, want=2",
			len(args))
	}

	mapper, _ := object.Cast(args[0])

	actualMap := &object.Map{}
	switch hashMap := mapper.(type) {
	case *object.Map:
		// do something
		actualMap = hashMap
	default:
		return newError("second arg needs to be a map in equals function")
	}

	newKey, _ := object.Cast(args[1])

	// key needs to implement Hashable interface
	hashKey, ok := newKey.(object.Hashable)
	if !ok {
		return newError("unusable as hash key: %s, consider one of those types (boolean, integer, float, string)", newKey.Type())
	}

	value, ok := actualMap.Pairs[hashKey.HashKey()]

	if !ok {
		return newError("no value found with associated hash key: %s", newKey.Inspect())
	}

	// actual value
	return value.Value
}

func DELETE(args ...object.Object) object.Object {
	if len(args) != 2 {
		return newError("wrong number of arguments. got=%d, want=2",
			len(args))
	}

	if args[0].Type() != object.MAP_OBJ {
		return newError("first arg needs to be of type map, got %v", args[0].Type())
	}

	arg, isMutable := object.Cast(args[0])

	if !isMutable {
		return newError("provided map isn't mutable, since it was defined as a const, consider changing it to a mutable var")
	}

	args[1], _ = object.Cast(args[1])

	key, ok := args[1].(object.Hashable)

	if !ok {
		return newError("provided value doesn't can't be used as key")
	}

	mapp := arg.(*object.Map)

	_, ok = mapp.Pairs[key.HashKey()]

	if !ok {
		return newError("key (%v) is not associated with any value", key)
	}

	delete(mapp.Pairs, key.HashKey())

	return mapp
}
