package object

import (
	"blk/ast"
	"bytes"
	"fmt"
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
	BUILTIN_OBJ      = "BUILTIN"
	IMPORT_OBJ       = "IMPORT"

	// errors
	ERROR_OBJ = "ERROR"
)

type Object interface {
	Type() ObjectType
	Inspect() string
}

type Integer struct {
	Value int64
}

func (i *Integer) Type() ObjectType { return INTEGER_OBJ }
func (i *Integer) Inspect() string  { return fmt.Sprintf("%d", i.Value) }

type Boolean struct {
	Value bool
}

func (b *Boolean) Type() ObjectType { return BOOLEAN_OBJ }
func (b *Boolean) Inspect() string  { return fmt.Sprintf("%t", b.Value) }

type Float struct {
	Value float64
}

func (b *Float) Type() ObjectType { return FLOAT_OBJ }
func (b *Float) Inspect() string  { return fmt.Sprintf("%f", b.Value) }

type String struct {
	Value string
}

func (b *String) Type() ObjectType { return STRING_OBJ }
func (b *String) Inspect() string  { return b.Value }

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
