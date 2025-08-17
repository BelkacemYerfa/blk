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
	CHAR_OBJ            = "CHAR"
	NUL_OBJ             = "NUL"
	RETURN_VALUE_OBJ    = "RETURN_VALUE"
	FUNCTION_OBJ        = "FUNCTION"
	IMPORT_OBJ          = "IMPORT"
	ARRAY_OBJ           = "ARRAY"
	MAP_OBJ             = "MAP"
	SKIP_OBJ            = "SKIP"
	BREAK_OBJ           = "BREAK"
	STRUCT_OBJ          = "STRUCT"
	STRUCT_INSTANCE_OBJ = "STRUCT_INSTANCE"
	BUILTIN_MODULE      = "BUILTIN_MODULE"
	USER_MODULE         = "USER_MODULE"
	BUILTIN_OBJ         = "BUILTIN"

	// errors
	ERROR_OBJ = "ERROR"
)

type HashKey struct {
	Type  ObjectType
	Value float64
}

type Object interface {

	// returns the object type, check the prefix_OBJ const above
	Type() ObjectType

	// inspects the value that is current object has
	Inspect() string

	// create a deep copy off the current value that the object has
	Copy() Object

	// checks wether 2 objects are equals or not
	Equals(other Object) bool
}

type EmptyObjImplementation struct{}

func (i *EmptyObjImplementation) Type() ObjectType {
	panic("Not Implemented")
}

func (i *EmptyObjImplementation) Inspect() string {
	panic("Not Implemented")
}

func (i *EmptyObjImplementation) Copy() Object {
	panic("Not Implemented")
}

func (i *EmptyObjImplementation) Equals(other Object) bool {
	panic("Not Implemented")
}

type Integer struct {
	EmptyObjImplementation
	Value int64
}

func (i *Integer) Type() ObjectType { return INTEGER_OBJ }
func (i *Integer) Inspect() string  { return fmt.Sprintf("%d", i.Value) }
func (i *Integer) Copy() Object {
	return &Integer{
		Value: i.Value,
	}
}
func (i *Integer) Equals(v Object) bool {
	bVal, ok := v.(*Integer)
	return ok && i.Value == bVal.Value
}
func (i *Integer) HashKey() HashKey {
	return HashKey{Type: i.Type(), Value: float64(i.Value)}
}

type Boolean struct {
	EmptyObjImplementation
	Value bool
}

func (b *Boolean) Type() ObjectType { return BOOLEAN_OBJ }
func (b *Boolean) Inspect() string  { return fmt.Sprintf("%t", b.Value) }
func (i *Boolean) Copy() Object {
	return &Boolean{
		Value: i.Value,
	}
}
func (i *Boolean) Equals(v Object) bool {
	bVal, ok := v.(*Boolean)
	return ok && i.Value == bVal.Value
}
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
	EmptyObjImplementation
	Value float64
}

func (b *Float) Type() ObjectType { return FLOAT_OBJ }
func (b *Float) Inspect() string  { return fmt.Sprintf("%f", b.Value) }
func (i *Float) Copy() Object {
	return &Float{
		Value: i.Value,
	}
}
func (i *Float) Equals(v Object) bool {
	bVal, ok := v.(*Float)
	return ok && i.Value == bVal.Value
}
func (b *Float) HashKey() HashKey {
	return HashKey{Type: b.Type(), Value: b.Value}
}

type String struct {
	EmptyObjImplementation
	Value string
}

func (b *String) Type() ObjectType { return STRING_OBJ }
func (b *String) Inspect() string  { return b.Value }
func (i *String) Copy() Object {
	return &String{
		Value: i.Value,
	}
}
func (i *String) Equals(v Object) bool {
	bVal, ok := v.(*String)
	return ok && i.Value == bVal.Value
}
func (s *String) HashKey() HashKey {
	h := fnv.New64a()
	h.Write([]byte(s.Value))
	return HashKey{Type: s.Type(), Value: float64(h.Sum64())}
}

type Char struct {
	EmptyObjImplementation
	Value rune
}

func (b *Char) Type() ObjectType { return CHAR_OBJ }
func (b *Char) Inspect() string  { return string(b.Value) }
func (i *Char) Copy() Object {
	return &Char{
		Value: i.Value,
	}
}
func (i *Char) Equals(v Object) bool {
	bVal, ok := v.(*Char)
	return ok && i.Value == bVal.Value
}
func (s *Char) HashKey() HashKey {
	h := fnv.New64a()
	h.Write([]byte(string(s.Value)))
	return HashKey{Type: s.Type(), Value: float64(h.Sum64())}
}

type Nul struct {
	EmptyObjImplementation
}

func (b *Nul) Type() ObjectType { return NUL_OBJ }
func (b *Nul) Inspect() string  { return "nul" }
func (i *Nul) Copy() Object     { return i }

type ReturnValue struct {
	EmptyObjImplementation
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
	EmptyObjImplementation
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
	EmptyObjImplementation
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
func (i *Array) Equals(v Object) bool {
	bVal, ok := v.(*Array)
	if !ok {
		return false
	}
	if len(bVal.Elements) != len(i.Elements) {
		return false
	}
	for idx, elem := range bVal.Elements {
		value := i.Elements[idx]
		if !elem.Equals(value) {
			return false
		}
	}
	return true
}
func (i *Array) Copy() Object {
	elements := make([]Object, 0)

	for _, v := range i.Elements {
		elements = append(elements, v.Copy())
	}

	return &Array{
		Size:     i.Size,
		Elements: elements,
	}
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
	EmptyObjImplementation
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

func (i *Map) Copy() Object {
	pairs := make(PairsType, 0)

	for k, v := range i.Pairs {
		pairs[k] = HashPair{
			Key:   v.Key.Copy(),
			Value: v.Value.Copy(),
		}
	}

	return &Map{
		Pairs: pairs,
	}
}

func (i *Map) Equals(v Object) bool {
	bVal, ok := v.(*Map)
	if !ok {
		return false
	}
	if len(bVal.Pairs) != len(i.Pairs) {
		return false
	}
	for key, elem := range bVal.Pairs {
		value, ok := i.Pairs[key]
		if !ok {
			return false
		}
		if !elem.Key.Equals(value.Key) || elem.Value.Equals(value.Value) {
			return false
		}
	}
	return true
}

type Error struct {
	EmptyObjImplementation
	Message string
}

func (e *Error) Type() ObjectType { return ERROR_OBJ }
func (e *Error) Inspect() string  { return e.Message }
func (e *Error) Copy() Object     { return e }

type BuiltinFunction func(args ...Object) Object

type BuiltinFn struct {
	EmptyObjImplementation
	Fn BuiltinFunction
}

func (b *BuiltinFn) Type() ObjectType { return BUILTIN_OBJ }
func (b *BuiltinFn) Inspect() string  { return "builtin function" }

type BuiltinConst struct {
	EmptyObjImplementation
	Const Object
}

func (b *BuiltinConst) Type() ObjectType { return BUILTIN_OBJ }
func (b *BuiltinConst) Inspect() string  { return b.Const.Inspect() }

// this for module type which can be constants, functions (for now)
type Module = map[string]Object

// proper module import with namespaces
type BuiltInModule struct {
	EmptyObjImplementation
	Name  string
	Attrs map[string]Object
}

func (b *BuiltInModule) Type() ObjectType { return BUILTIN_MODULE }

// TODO: update this method later
func (b *BuiltInModule) Inspect() string { return b.Name }

// user module, another file
// TODO: structure to use for user modules
type UserModule struct {
	EmptyObjImplementation
	Name  string
	Attrs map[string]Object
}

func (b *UserModule) Type() ObjectType { return USER_MODULE }

func (b *UserModule) Inspect() string { return b.Name }

type Skip struct {
	EmptyObjImplementation
}

func (b *Skip) Type() ObjectType { return SKIP_OBJ }
func (b *Skip) Inspect() string  { return "skip" }

type Break struct {
	EmptyObjImplementation
}

func (b *Break) Type() ObjectType { return BREAK_OBJ }
func (b *Break) Inspect() string  { return "skip" }

type Struct struct {
	EmptyObjImplementation
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

func (i *Struct) Copy() Object {
	strct := &Struct{
		Fields: make(map[string]Object),
	}

	for k, v := range i.Fields {
		strct.Fields[k] = v.Copy()
	}

	strct.Methods = i.Methods

	return strct
}

type StructInstance struct {
	EmptyObjImplementation
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

func (i *StructInstance) Copy() Object {
	strct := &StructInstance{
		Fields: make(map[string]Object),
	}

	for k, v := range i.Fields {
		strct.Fields[k] = v.Copy()
	}

	strct.Methods = i.Methods

	return strct
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
	case *Nul:
		return true
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

func UseCopyValueOrRef(obj Object) Object {
	obj, _ = Cast(obj)
	switch v := obj.(type) {
	// means that this types are give u a deep copy of their value
	case *Float, *Integer, *String, *Boolean:
		return v.Copy()

	// means that this types are being shallow copied
	case *Array, *Map, *Struct, *StructInstance:
		return v

	default:
		return v
	}
}
