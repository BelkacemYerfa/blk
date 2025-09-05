package stdlib

import (
	"blk/object"
	"slices"
	"sort"
)

var arrayModule = object.Module{
	"equals":  &object.BuiltinFn{Fn: arrayEquals},
	"index":   &object.BuiltinFn{Fn: arrayIndex},
	"append":  &object.BuiltinFn{Fn: arrayAppend},
	"reverse": &object.BuiltinFn{Fn: arrayReverse},
	"sort":    &object.BuiltinFn{Fn: arraySort},
	"min":     &object.BuiltinFn{Fn: arrayMin},
	"max":     &object.BuiltinFn{Fn: arrayMax},
	"replace": &object.BuiltinFn{Fn: arrayReplace},
	"insert":  &object.BuiltinFn{Fn: arrayInsert},
	"delete":  &object.BuiltinFn{Fn: arrayDelete},
	"concat":  &object.BuiltinFn{Fn: arrayConcat},
	// "contains": &object.BuiltinFn{Fn: arrayContains},
}

// checks if 2 arrays are equals or not, and this by using a builtin method on the object interface
// results are
// if length of array a, b is different false will get returned
// if at least one single value is different false will get returned
// otherwise true
// usage:
// -	equals := array.equals(users, authors)
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

// returns the index of an element in an array if it exists, if not -1 will get returned
// usage:
// -	index := array.index(users, "John Doe")
func arrayIndex(args ...object.Object) object.Object {
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

// takes an array & an element, append the element to the end of the array
// if the array is a fixed size array & the length == fixed_size, and error will get returned
// usage:
// -	array.append(users, "John Doe")
func arrayAppend(args ...object.Object) object.Object {
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
		if actualArray.Size == len(actualArray.Elements) && actualArray.Size > 0 {
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

// takes an array, and reverse the order of its elements
// usage:
// -	array.reverse(users)
func arrayReverse(args ...object.Object) object.Object {
	if len(args) != 1 {
		return newError("wrong number of arguments. got=%d, want=1",
			len(args))
	}
	arg, isMutable := object.Cast(args[0])

	if arg.Type() != object.ARRAY_OBJ {
		return newError("argument needs to be of type array, got %v", arg.Type())
	}

	if !isMutable {
		return newError("can't mutate %v, probably defined as a const", args[0].Inspect())
	}

	array := arg.(*object.Array)

	slices.Reverse(array.Elements)

	return array
}

func sortObjects(elems []object.Object) {
	sort.Slice(elems, func(i, j int) bool {
		switch vi := elems[i].(type) {
		case *object.Integer:
			vj := elems[j].(*object.Integer)
			return vi.Value < vj.Value
		case *object.Float:
			vj := elems[j].(*object.Float)
			return vi.Value < vj.Value
		case *object.String:
			vj := elems[j].(*object.String)
			return vi.Value < vj.Value
		default:
			return false
		}
	})
}

// takes an array, and sorts it
// but the elements of the array need to be of one of this types (string, int, float)
// usage:
// -	array.sort(users)
func arraySort(args ...object.Object) object.Object {
	if len(args) != 1 {
		return newError("wrong number of arguments. got=%d, want=1",
			len(args))
	}
	arg, isMutable := object.Cast(args[0])

	if arg.Type() != object.ARRAY_OBJ {
		return newError("argument needs to be of type array, got %v", arg.Type())
	}

	if !isMutable {
		return newError("can't mutate %v, probably defined as a const", args[0].Inspect())
	}

	array := arg.(*object.Array)
	acceptedTypes := []object.ObjectType{
		object.STRING_OBJ, object.INTEGER_OBJ, object.FLOAT_OBJ,
	}

	// check that element type is one of this for sorting
	// string, int, float
	if !slices.Contains(acceptedTypes, array.Elements[0].Type()) {
		return newError("sort method, only works on strings, integers and floats, provided element type %v", array.Elements[0].Type())
	}

	sortObjects(array.Elements)

	return array
}

// takes an array, and returns the min value in the array
// but the elements of the array need to be of one of this types (string, int, float)
// usage:
// -	min := array.min(users)
func arrayMin(args ...object.Object) object.Object {
	if len(args) != 1 {
		return newError("wrong number of arguments. got=%d, want=1",
			len(args))
	}
	arg, _ := object.Cast(args[0])

	if arg.Type() != object.ARRAY_OBJ {
		return newError("argument needs to be of type array, got %v", arg.Type())
	}

	array := arg.(*object.Array)
	acceptedTypes := []object.ObjectType{
		object.STRING_OBJ, object.INTEGER_OBJ, object.FLOAT_OBJ,
	}

	// check that element type is one of this for sorting
	// string, int, float
	if !slices.Contains(acceptedTypes, array.Elements[0].Type()) {
		return newError("min method, only works on strings, integers and floats, provided element type %v", array.Elements[0].Type())
	}

	sortObjects(array.Elements)

	return array.Elements[0]
}

// takes an array, and returns the max value in the array
// but the elements of the array need to be of one of this types (string, int, float)
// usage:
// -	max := array.max(users)
func arrayMax(args ...object.Object) object.Object {
	if len(args) != 1 {
		return newError("wrong number of arguments. got=%d, want=1",
			len(args))
	}
	arg, _ := object.Cast(args[0])

	if arg.Type() != object.ARRAY_OBJ {
		return newError("argument needs to be of type array, got %v", arg.Type())
	}

	array := arg.(*object.Array)
	acceptedTypes := []object.ObjectType{
		object.STRING_OBJ, object.INTEGER_OBJ, object.FLOAT_OBJ,
	}

	// check that element type is one of this for sorting
	// string, int, float
	if !slices.Contains(acceptedTypes, array.Elements[0].Type()) {
		return newError("max method, only works on strings, integers and floats, provided element type %v", array.Elements[0].Type())
	}

	sortObjects(array.Elements)

	return array.Elements[len(array.Elements)-1]
}

// takes an array, start, end index, and variadic of arguments
// and replaces the elements from start to end with the new provided elements,
// for fixed size arrays if the operation excide the defined size, an error will get returned
// if end > len(array), an error will get returned
// if start > end, an error will get returned
// usage:
// -	array.replace(ids, 1, 3, "0xA15", "0xD72")
func arrayReplace(args ...object.Object) object.Object {
	if len(args) < 4 {
		return newError("wrong number of arguments. got=%d, at least = 4",
			len(args))
	}
	arg, isMutable := object.Cast(args[0])

	if arg.Type() != object.ARRAY_OBJ {
		return newError("argument needs to be of type array, got %v", arg.Type())
	}

	if !isMutable {
		return newError("can't mutate %v, probably defined as a const", args[0].Inspect())
	}

	array := arg.(*object.Array)

	// check the i,k arguments
	args[1], _ = object.Cast(args[1])
	args[2], _ = object.Cast(args[2])

	if args[1].Type() != object.INTEGER_OBJ || args[2].Type() != object.INTEGER_OBJ {
		return newError("both start & end of replace arguments, need to be of type integer")
	}

	// check that start <= end
	start := args[1].(*object.Integer).Value
	end := args[2].(*object.Integer).Value

	if start > end {
		return newError("start <= end always, instead we got start > end")
	}

	// meant for fixed size arrays
	// check the end and the size of the array
	if int(end) > len(array.Elements) {
		return newError("end > len of the array, consider adjusting the end argument")
	}

	// check the variadic params left are of the same type of the array type
	variadic := &object.Array{
		Size:     len(args[3:]),
		Elements: args[3:],
	}

	// type check first on the variadicElems
	// no need to check the length
	if !object.ObjectTypesCheck(array, variadic, false) {
		return newError("variadic arguments provided need to be of same type, the current array has")
	}

	temp := slices.Replace(array.Elements, int(start), int(end), variadic.Elements...)

	// for fixed size arrays
	if len(temp) > array.Size && array.Size > 0 {
		return newError("operation will result length will excide the array size set, consider adjusting the array size, or changing the number of values provided")
	}

	array.Elements = temp

	return array
}

// takes an array, index and variadic of arguments
// and inserts the elements from index up until the len(variadic)+index
// for fixed size arrays if the operation excide the defined size, an error will get returned
// if index > len(array), an error will get returned
// usage:
// -	array.insert(ids, 1, "0xA15", "0xD72")
func arrayInsert(args ...object.Object) object.Object {
	if len(args) < 3 {
		return newError("wrong number of arguments. got=%d, at least = 3",
			len(args))
	}
	arg, isMutable := object.Cast(args[0])

	if arg.Type() != object.ARRAY_OBJ {
		return newError("argument needs to be of type array, got %v", arg.Type())
	}

	if !isMutable {
		return newError("can't mutate %v, probably defined as a const", args[0].Inspect())
	}

	array := arg.(*object.Array)

	// check the i,k arguments
	args[1], _ = object.Cast(args[1])

	if args[1].Type() != object.INTEGER_OBJ {
		return newError("index of insert, needs to be of type integer")
	}

	// check that start <= end
	index := args[1].(*object.Integer).Value

	// meant for fixed size arrays
	// check the end and the size of the array
	if int(index) > len(array.Elements) {
		return newError("index > len of the array, consider changing the initial array size, or adjust the end argument")
	}

	// check the variadic params left are of the same type of the array type
	variadic := &object.Array{
		Size:     len(args[2:]),
		Elements: args[2:],
	}

	// type check first on the variadicElems
	// no need to check the length
	if !object.ObjectTypesCheck(array, variadic, false) {
		return newError("variadic arguments provided need to be of same type, the current array has")
	}

	temp := slices.Insert(array.Elements, int(index), variadic.Elements...)

	// for fixed size arrays
	if len(temp) > array.Size && array.Size > 0 {
		return newError("operation will result length will excide the array size set, consider adjusting the array size, or changing the number of values provided")
	}

	array.Elements = temp

	return array
}

// takes an array, start and end index
// then it deletes the elements from start up until end
// if end > len(array), an error will get returned
// if start > end, an error will get returned
// usage:
// -	array.delete(ids, 3, 5)
func arrayDelete(args ...object.Object) object.Object {
	if len(args) != 3 {
		return newError("wrong number of arguments. got=%d, want= 3",
			len(args))
	}
	arg, isMutable := object.Cast(args[0])

	if arg.Type() != object.ARRAY_OBJ {
		return newError("argument needs to be of type array, got %v", arg.Type())
	}

	if !isMutable {
		return newError("can't mutate %v, probably defined as a const", args[0].Inspect())
	}

	array := arg.(*object.Array)

	// check the i,k arguments
	args[1], _ = object.Cast(args[1])
	args[2], _ = object.Cast(args[2])

	if args[1].Type() != object.INTEGER_OBJ || args[2].Type() != object.INTEGER_OBJ {
		return newError("both start & end of replace arguments, need to be of type integer")
	}

	// check that start <= end
	start := args[1].(*object.Integer).Value
	end := args[2].(*object.Integer).Value

	if start > end {
		return newError("start <= end always, instead we got start > end")
	}

	// meant for fixed size arrays
	// check the end and the size of the array
	if int(end) > len(array.Elements) {
		return newError("end > size of the array, consider adjusting the end argument")
	}

	array.Elements = slices.Delete(array.Elements, int(start), int(end))

	return array
}

// takes a set of arrays, merges them and returns the new array
// note that all of the arrays elements need to be of the same type
// if not an error will get returned
// usage:
// -	newArray := array.concat(users, names, people)
func arrayConcat(args ...object.Object) object.Object {
	if len(args) < 2 {
		return newError("wrong number of arguments. got=%d, at least = 2",
			len(args))
	}
	arg, _ := object.Cast(args[0])

	if arg.Type() != object.ARRAY_OBJ {
		return newError("argument needs to be of type array, got %v", arg.Type())
	}

	array := arg.(*object.Array)

	variadic := args[1:]

	temp := append([]object.Object{array.Elements[0]}, array.Elements[1:]...)

	// check the variadic params left are of the same type of the array type
	for _, arr := range variadic {
		arr, _ := object.Cast(arr)
		if arr.Type() != object.ARRAY_OBJ {
			return newError("all variadic arguments have to be of type array")
		}
		if !object.ObjectTypesCheck(array, arr, false) {
			return newError("variadic arguments provided need to be of same type, the current array has")
		}
		temp = append(temp, arr.(*object.Array).Elements...)
	}

	return &object.Array{
		Elements: temp,
	}
}

// func arrayContains(args ...object.Object) object.Object {
// 	if len(args) != 2 {
// 		return newError("wrong number of arguments. got=%d, want=2",
// 			len(args))
// 	}
// 	arg, _ := object.Cast(args[0])

// 	if arg.Type() != object.ARRAY_OBJ {
// 		return newError("argument needs to be of type array, got %v", arg.Type())
// 	}

// 	array := arg.(*object.Array)

// 	acceptedTypes := []object.ObjectType{
// 		object.STRING_OBJ, object.INTEGER_OBJ, object.FLOAT_OBJ,
// 	}

// 	// check that element type is one of this for sorting
// 	// string, int, float
// 	if !slices.Contains(acceptedTypes, array.Elements[0].Type()) {
// 		return newError("max method, only works on strings, integers and floats, provided element type %v", array.Elements[0].Type())
// 	}

// 	arg2, _ := object.Cast(args[2])

// 	if !slices.Contains(acceptedTypes, arg )

// 	_, found := slices.BinarySearchFunc(array.Elements)

// 	return found
// }
