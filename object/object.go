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
	INTEGER_OBJ      = "INTEGER"
	BOOLEAN_OBJ      = "BOOLEAN"
	FLOAT_OBJ        = "FLOAT"
	STRING_OBJ       = "STRING"
	RETURN_VALUE_OBJ = "RETURN_VALUE"
	FUNCTION_OBJ     = "FUNCTION"
	IMPORT_OBJ       = "IMPORT"
	ARRAY_OBJ        = "ARRAY"
	MAP_OBJ          = "MAP"
	STRUCT_OBJ       = "STRUCT"
	BUILTIN_MODULE   = "BUILTIN_MODULE"
	BUILTIN_OBJ      = "BUILTIN"

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
	Value Object
}

func (rv *ReturnValue) Type() ObjectType { return RETURN_VALUE_OBJ }
func (rv *ReturnValue) Inspect() string  { return rv.Value.Inspect() }

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
func (e *Error) Inspect() string  { return "ERROR: " + e.Message }

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

type Struct struct {
	// Attrs are both variable and methods
	Attrs map[string]Object
}

func (b *Struct) Type() ObjectType { return STRUCT_OBJ }

// TODO: update this method later
func (b *Struct) Inspect() string { return "am struct" }

func Cast(obj Object) (Object, bool) {
	switch obj := obj.(type) {
	case ItemObject:
		return obj.Object, obj.IsMutable
	case *ItemObject:
		return obj.Object, obj.IsMutable
	default:
		return obj, false
	}
}

func ObjectEquals(a, b Object) bool {
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
	// TODO: needs to be extended for recursive compare later
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
		for i, elem := range val.Elements {
			elements[i] = DeepCopy(elem)
		}
		return &Array{Elements: elements}
	case *Map:
		pairs := make(PairsType, len(val.Pairs))
		for i, elem := range val.Pairs {
			pairs[i] = HashPair{
				Key:   DeepCopy(elem.Key),
				Value: DeepCopy(elem.Value),
			}
		}
		return &Map{Pairs: pairs}
	// Add other types as needed...
	default:
		return val // For immutable or not-clonable types (like Error, etc.)
	}
}
