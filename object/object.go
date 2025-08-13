package object

import (
	"blk/ast"
	"bytes"
	"fmt"
	"hash/fnv"
	"strings"
)

type ObjectType string

const (
	INTEGER_OBJ         = "INTEGER"
	BOOLEAN_OBJ         = "BOOLEAN"
	FLOAT_OBJ           = "FLOAT"
	STRING_OBJ          = "STRING"
	RETURN_VALUE_OBJ    = "RETURN_VALUE"
	FUNCTION_OBJ        = "FUNCTION"
	IMPORT_OBJ          = "IMPORT"
	ARRAY_OBJ           = "ARRAY"
	MAP_OBJ             = "MAP"
	SKIP_OBJ            = "SKIP"
	STRUCT_OBJ          = "STRUCT"
	STRUCT_INSTANCE_OBJ = "STRUCT_INSTANCE"
	BUILTIN_MODULE      = "BUILTIN_MODULE"
	BUILTIN_OBJ         = "BUILTIN"

	// errors
	ERROR_OBJ = "ERROR"
)

type HashKey struct {
	Type  ObjectType
	Value float64
}

type Object interface {
	Type() ObjectType
	Inspect() string
}

type Integer struct {
	Value int64
}

func (i *Integer) Type() ObjectType { return INTEGER_OBJ }
func (i *Integer) Inspect() string  { return fmt.Sprintf("%d", i.Value) }
func (i *Integer) HashKey() HashKey {
	return HashKey{Type: i.Type(), Value: float64(i.Value)}
}

type Boolean struct {
	Value bool
}

func (b *Boolean) Type() ObjectType { return BOOLEAN_OBJ }
func (b *Boolean) Inspect() string  { return fmt.Sprintf("%t", b.Value) }
func (b *Boolean) HashKey() HashKey {
	var value uint64
	if b.Value {
		value = 1
	} else {
		value = 0
	}
	return HashKey{Type: b.Type(), Value: float64(value)}
}

type Float struct {
	Value float64
}

func (b *Float) Type() ObjectType { return FLOAT_OBJ }
func (b *Float) Inspect() string  { return fmt.Sprintf("%f", b.Value) }
func (b *Float) HashKey() HashKey {
	return HashKey{Type: b.Type(), Value: b.Value}
}

type String struct {
	Value string
}

func (b *String) Type() ObjectType { return STRING_OBJ }
func (b *String) Inspect() string  { return b.Value }
func (s *String) HashKey() HashKey {
	h := fnv.New64a()
	h.Write([]byte(s.Value))
	return HashKey{Type: s.Type(), Value: float64(h.Sum64())}
}

type ReturnValue struct {
	Values []Object
}

func (rv *ReturnValue) Type() ObjectType { return RETURN_VALUE_OBJ }
func (rv *ReturnValue) Inspect() string {
	var out bytes.Buffer
	out.WriteString("[")
	for idx, elem := range rv.Values {
		out.WriteString(elem.Inspect())
		if idx+1 <= len(rv.Values)-1 {
			out.WriteString(", ")
		}
	}
	out.WriteString("]")
	return out.String()
}

type Function struct {
	Parameters []*ast.Identifier
	Body       *ast.BlockStatement
	Env        *Environment
}

func (f *Function) Type() ObjectType { return FUNCTION_OBJ }
func (f *Function) Inspect() string {
	var out bytes.Buffer
	params := []string{}
	for _, p := range f.Parameters {
		params = append(params, p.String())
	}
	out.WriteString("fn")
	out.WriteString("(")
	out.WriteString(strings.Join(params, ", "))
	out.WriteString(") {\n")
	out.WriteString(f.Body.String())
	out.WriteString("\n}")
	return out.String()
}

type Array struct {
	Size     int // if size == -1 means that the array is dynamic
	Elements []Object
}

func (a *Array) Type() ObjectType { return ARRAY_OBJ }
func (a *Array) Inspect() string {
	var out bytes.Buffer
	out.WriteString("[")
	for idx, elem := range a.Elements {
		out.WriteString(elem.Inspect())
		if idx+1 <= len(a.Elements)-1 {
			out.WriteString(", ")
		}
	}
	out.WriteString("]")
	return out.String()
}

type Hashable interface {
	HashKey() HashKey
}

type HashPair struct {
	Key   Object
	Value Object
}

type PairsType = map[HashKey]HashPair

type Map struct {
	Pairs PairsType
}

func (a *Map) Type() ObjectType { return MAP_OBJ }
func (a *Map) Inspect() string {
	var out bytes.Buffer
	pairs := []string{}
	for _, pair := range a.Pairs {
		pairs = append(pairs, fmt.Sprintf("%s: %s",
			pair.Key.Inspect(), pair.Value.Inspect()))
	}
	out.WriteString("{")
	out.WriteString(strings.Join(pairs, ", "))
	out.WriteString("}")
	return out.String()
}

type Error struct {
	Message string
}

func (e *Error) Type() ObjectType { return ERROR_OBJ }
func (e *Error) Inspect() string  { return e.Message }

type BuiltinFunction func(args ...Object) Object

type BuiltinFn struct {
	Fn BuiltinFunction
}

func (b *BuiltinFn) Type() ObjectType { return BUILTIN_OBJ }
func (b *BuiltinFn) Inspect() string  { return "builtin function" }

type BuiltinConst struct {
	Const Object
}

func (b *BuiltinConst) Type() ObjectType { return BUILTIN_OBJ }
func (b *BuiltinConst) Inspect() string  { return b.Const.Inspect() }

// this for module type which can be constants, functions (for now)
type Module = map[string]Object

// proper module import with namespaces
type BuiltInModule struct {
	Name  string
	Attrs map[string]Object
}

func (b *BuiltInModule) Type() ObjectType { return BUILTIN_MODULE }

// TODO: update this method later
func (b *BuiltInModule) Inspect() string { return b.Name }

type Skip struct{}

func (b *Skip) Type() ObjectType { return SKIP_OBJ }
func (b *Skip) Inspect() string  { return "skip" }

type Struct struct {
	// Fields are both variable decl
	Fields map[string]Object
	// Methods are the builtin function that u can use from the struct
	Methods map[string]Object
}

func (b *Struct) Type() ObjectType { return STRUCT_OBJ }
func (b *Struct) Inspect() string {
	var out bytes.Buffer
	out.WriteString("struct {")
	for name, field := range b.Fields {
		out.WriteString(name + " := " + field.Inspect())
	}
	for name, function := range b.Methods {
		out.WriteString(name + " : " + function.Inspect())
	}
	out.WriteString("}")
	return out.String()
}

type StructInstance struct {
	// Fields are both variable decl
	Fields map[string]Object
	// Methods are the builtin function that u can use from the struct
	Methods map[string]Object
}

func (b *StructInstance) Type() ObjectType { return STRUCT_INSTANCE_OBJ }
func (b *StructInstance) Inspect() string {
	var out bytes.Buffer
	out.WriteString("struct {")
	for name, field := range b.Fields {
		out.WriteString(name + " := " + field.Inspect())
	}
	for name, function := range b.Methods {
		out.WriteString(name + " : " + function.Inspect())
	}
	out.WriteString("}")
	return out.String()
}

func Cast(obj Object) (Object, bool) {
	switch obj := obj.(type) {
	case ItemObject:
		return obj.Object, obj.IsMutable
	case *ItemObject:
		return obj.Object, obj.IsMutable
	default:
		return obj, true
	}
}

func ObjectEquals(a, b Object) bool {
	a, _ = Cast(a)
	b, _ = Cast(b)
	switch aVal := a.(type) {
	case *Integer:
		bVal, ok := b.(*Integer)
		return ok && aVal.Value == bVal.Value
	case *Boolean:
		bVal, ok := b.(*Boolean)
		return ok && aVal.Value == bVal.Value
	case *String:
		bVal, ok := b.(*String)
		return ok && aVal.Value == bVal.Value
	case *Float:
		bVal, ok := b.(*Float)
		return ok && aVal.Value == bVal.Value
	case *Array:
		bVal, ok := b.(*Array)
		if !ok {
			return false
		}
		if len(bVal.Elements) != len(aVal.Elements) {
			return false
		}
		for idx, elem := range bVal.Elements {
			value := aVal.Elements[idx]
			if !ObjectEquals(elem, value) {
				return false
			}
		}
		return true
	case *Map:
		bVal, ok := b.(*Map)
		if !ok {
			return false
		}
		if len(bVal.Pairs) != len(aVal.Pairs) {
			return false
		}
		for key, elem := range bVal.Pairs {
			value, ok := aVal.Pairs[key]
			if !ok {
				return false
			}
			if !ObjectEquals(elem.Key, value.Key) {
				return false
			}
			if !ObjectEquals(elem.Value, value.Value) {
				return false
			}
		}
		return true
	// struct maybe something here
	default:
		// fallback: not equal
		return false
	}
}

// build an equals method to all of the object type representation
func ObjectTypesCheck(a, b Object) bool {
	a, _ = Cast(a)
	b, _ = Cast(b)
	switch aVal := a.(type) {
	case *Integer:
		_, ok := b.(*Integer)
		return ok
	case *Boolean:
		_, ok := b.(*Boolean)
		return ok
	case *String:
		_, ok := b.(*String)
		return ok
	case *Float:
		_, ok := b.(*Float)
		return ok
	case *Array:
		bVal, ok := b.(*Array)
		if !ok {
			return false
		}
		if len(bVal.Elements) != len(aVal.Elements) {
			return false
		}
		for idx, elem := range bVal.Elements {
			value := aVal.Elements[idx]
			if !ObjectTypesCheck(elem, value) {
				return false
			}
		}
		return true
	case *Map:
		bVal, ok := b.(*Map)
		if !ok {
			return false
		}
		if len(bVal.Pairs) != len(aVal.Pairs) {
			return false
		}
		for key, elem := range bVal.Pairs {
			value, ok := aVal.Pairs[key]
			if !ok {
				return false
			}
			if !ObjectTypesCheck(elem.Key, value.Key) {
				return false
			}
			if !ObjectTypesCheck(elem.Value, value.Value) {
				return false
			}
		}
		return true
	case *Struct, *StructInstance:
		if a.Type() != b.Type() {
			return false
		}

		var bFields map[string]Object

		switch bTyped := b.(type) {
		case *Struct:
			bFields = bTyped.Fields
		case *StructInstance:
			bFields = bTyped.Fields
		default:
			return false
		}

		var aFields map[string]Object
		switch aTyped := aVal.(type) {
		case *Struct:
			aFields = aTyped.Fields
		case *StructInstance:
			aFields = aTyped.Fields
		}

		if len(bFields) != len(aFields) {
			return false
		}
		for k, v := range bFields {
			val, ok := aFields[k]
			if !ok {
				return false
			}
			if !ObjectTypesCheck(v, val) {
				return false
			}
		}

		return true
	default:
		// fallback: not equal
		return false
	}
}

func DeepCopy(obj Object) Object {
	switch val := obj.(type) {
	case *Integer:
		return &Integer{Value: val.Value}
	case *String:
		return &String{Value: val.Value}
	case *Boolean:
		return &Boolean{Value: val.Value}
	case *Float:
		return &Float{Value: val.Value}
	case *Array:
		elements := make([]Object, 0, len(val.Elements))
		for _, elem := range val.Elements {
			cast, _ := Cast(elem)
			elements = append(elements, cast)
		}
		return &Array{Elements: elements}
	case *Map:
		pairs := make(PairsType, len(val.Pairs))
		for i, elem := range val.Pairs {
			key, _ := Cast(elem.Key)
			value, _ := Cast(elem.Value)
			pairs[i] = HashPair{
				Key:   key,
				Value: value,
			}
		}
		return &Map{Pairs: pairs}
	case *Struct, *StructInstance:
		var fields map[string]Object
		var methods map[string]Object

		switch typed := val.(type) {
		case *Struct:
			fields = typed.Fields
			methods = typed.Methods
		case *StructInstance:
			fields = typed.Fields
			methods = typed.Methods
		}

		// Deep copy fields
		copiedFields := make(map[string]Object, len(fields))
		for k, v := range fields {
			cast, _ := Cast(v)
			copiedFields[k] = DeepCopy(cast)
		}

		// Return the same type as input
		switch val.(type) {
		case *Struct:
			return &Struct{
				Fields:  copiedFields,
				Methods: methods,
			}
		case *StructInstance:
			return &StructInstance{
				Fields:  copiedFields,
				Methods: methods,
			}
		}

		return nil
	default:
		return val // For immutable or not-clonable types (like Error, etc.)
	}
}

func UseCopyValueOrRef(obj Object) Object {
	obj, _ = Cast(obj)
	switch v := obj.(type) {
	// means that this types are give u a deep copy of their value
	case *Float, *Integer, *String, *Boolean:
		return DeepCopy(v)

	// means that this types are being shallow copied
	case *Array, *Map, *Struct, *StructInstance:
		return v

	default:
		return v
	}
}
