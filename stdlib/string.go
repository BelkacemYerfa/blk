package stdlib

import (
	"blk/object"
	"strings"
)

var stringModule = object.Module{
	"join":        &object.BuiltinFn{Fn: stringJoin},
	"split":       &object.BuiltinFn{Fn: stringSplit},
	"hasSuffix":   &object.BuiltinFn{Fn: funcSSB(strings.HasSuffix)},
	"hasPrefix":   &object.BuiltinFn{Fn: funcSSB(strings.HasPrefix)},
	"contains":    &object.BuiltinFn{Fn: funcSSB(strings.Contains)},
	"containsAny": &object.BuiltinFn{Fn: funcSSB(strings.ContainsAny)},
	"equalFold":   &object.BuiltinFn{Fn: funcSSB(strings.EqualFold)},
	"toUpperCase": &object.BuiltinFn{Fn: funcSS(strings.ToUpper)},
	"toLowerCase": &object.BuiltinFn{Fn: funcSS(strings.ToLower)},
}

func stringJoin(args ...object.Object) object.Object {
	if len(args) != 2 {
		return newError("wrong number of arguments. got=%d, want=2",
			len(args))
	}

	// first args is an array
	if args[0].Type() != object.ARRAY_OBJ {
		return newError("first arg needs to be of type array, got=%v", args[0].Type())
	}
	args0, _ := object.Cast(args[0])
	array := args0.(*object.Array)

	// check the type of elements in the array
	// needs to be of type string
	if array.Elements[0].Type() != object.STRING_OBJ {
		return newError("first arg needs to be of type array, got=%v", args[0].Type())
	}

	// second one is the separator which is a string
	if args[1].Type() != object.STRING_OBJ {
		return newError("separator needs to be of type string, got=%v", args[0].Type())
	}

	args1, _ := object.Cast(args[1])
	separator := args1.(*object.String)

	elements := make([]string, 0)

	for _, elem := range array.Elements {
		elements = append(elements, elem.Inspect())
	}

	return &object.String{
		Value: strings.Join(elements, separator.Value),
	}
}

func stringSplit(args ...object.Object) object.Object {
	if len(args) != 2 {
		return newError("wrong number of arguments. got=%d, want=2",
			len(args))
	}

	// first args is an array
	if args[0].Type() != object.STRING_OBJ {
		return newError("first arg needs to be of type array, got=%v", args[0].Type())
	}
	args0, _ := object.Cast(args[0])
	s := args0.(*object.String)

	// second one is the separator which is a string
	if args[1].Type() != object.STRING_OBJ {
		return newError("separator needs to be of type string, got=%v", args[0].Type())
	}

	args1, _ := object.Cast(args[1])
	separator := args1.(*object.String)

	elements := strings.Split(s.Value, separator.Value)

	array := make([]object.Object, 0)

	for _, elem := range elements {
		array = append(array, &object.String{
			Value: elem,
		})
	}

	return &object.Array{
		Elements: array,
	}
}
