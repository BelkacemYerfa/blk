package ast

import (
	"blk/lexer"
	"bytes"
	"fmt"
	"strings"
)

type Node interface {
	TokenLiteral() string
	String() string
	GetToken() lexer.Token
}

type Statement interface {
	Node
	statementNode()
}

type Expression interface {
	Node
	expressionNode()
}

type Program struct {
	Node
	Statements []Statement
}

func (p *Program) TokenLiteral() string {
	if len(p.Statements) > 0 {
		return p.Statements[0].TokenLiteral()
	} else {
		return ""
	}
}

func (p *Program) String() string {
	var out bytes.Buffer
	for idx, s := range p.Statements {
		out.WriteString(s.String())
		if idx < len(p.Statements)-1 {
			out.WriteString("\n")
		}
	}
	return out.String()
}

type TypeKind int

const (
	_ TypeKind = iota
	TypeVoid
	TypeInt8
	TypeInt16
	TypeInt32
	TypeInt64
	TypeUInt8
	TypeUInt16
	TypeUInt32
	TypeUInt64
	TypeFloat32
	TypeFloat64
	TypeChar
	TypeString
	TypeBool
	TypeAny
	TypeArray
	TypeMap
	TypePointer
	TypeEnum
	TypeStruct
	TypeFunction
	TypeAlias
)

var PrimitiveTypes = map[lexer.TokenKind]TypeKind{
	lexer.TokenString:  TypeString,
	lexer.TokenChar:    TypeChar,
	lexer.TokenBool:    TypeBool,
	lexer.TokenInt8:    TypeInt8,
	lexer.TokenInt16:   TypeInt16,
	lexer.TokenInt32:   TypeInt32,
	lexer.TokenInt64:   TypeInt64,
	lexer.TokenUInt8:   TypeUInt8,
	lexer.TokenUInt16:  TypeUInt16,
	lexer.TokenUInt32:  TypeUInt32,
	lexer.TokenUInt64:  TypeUInt64,
	lexer.TokenFloat32: TypeFloat32,
	lexer.TokenFloat64: TypeFloat64,
	lexer.TokenAny:     TypeAny,
}

type Type interface {
	Type() TypeKind
	String() string
}

type PrimitiveType struct {
	Token  lexer.Token // token type
	Kind   TypeKind
	Size   int  // optional
	Signed bool // for ints
}

func (p *PrimitiveType) Type() TypeKind {
	return p.Kind
}
func (p *PrimitiveType) String() string {
	var out bytes.Buffer
	if p.Size > 0 {
		if p.Kind == TypeFloat32 || p.Kind == TypeFloat64 {
			out.WriteString("f" + fmt.Sprint(p.Size))
		} else {
			if p.Signed {
				out.WriteString("i" + fmt.Sprint(p.Size))
			} else {
				out.WriteString("u" + fmt.Sprint(p.Size))
			}
		}
	} else {
		if p.Token.Kind == "str" {
			out.WriteString("string")
		} else {
			out.WriteString(p.Token.Kind)
		}
	}
	return out.String()
}

// represents arrays (fixed size arrays & dynamic ones) & hash maps
// arrays:
// - fixed size: array(<type>, size)
// - dynamic: array(<type>)
// maps: map(<key-type>, <value-type>)
type CompositeType struct {
	Token     lexer.Token
	Kind      TypeKind
	LeftType  Type // can represent any underline type
	RightType Type
	Size      *IntegerLiteral
}

func (ct *CompositeType) Type() TypeKind {
	return ct.Kind
}
func (ct *CompositeType) String() string {
	var out bytes.Buffer
	if ct.Kind == TypeArray {
		out.WriteString("[")
		if ct.Size.Value > 0 {
			out.WriteString(ct.Size.String())
		} else {
			out.WriteString("..")
		}
		out.WriteString("]")
		out.WriteString(ct.LeftType.String())
	} else {
		out.WriteString("map[")
		out.WriteString(ct.LeftType.String())
		out.WriteString("]")
		out.WriteString(ct.RightType.String())
	}

	return out.String()
}

type AliasType struct {
	Token lexer.Token
	Kind  TypeKind
	Alias *Identifier
}

func (ct *AliasType) Type() TypeKind {
	return ct.Kind
}
func (ct *AliasType) String() string {
	var out bytes.Buffer
	out.WriteString(ct.Alias.String())
	return out.String()
}

type FunctionSignature struct {
	Token  lexer.Token
	Kind   TypeKind
	Args   map[string]*Arg
	Return []Type // support for multi return types
}

// represents function types such as fn(i8,i8): f32
type FunctionType struct {
	Token  lexer.Token
	Kind   TypeKind
	Args   []Type
	Return []Type // support for multi return types
}

func (ft *FunctionType) Type() TypeKind {
	return ft.Kind
}
func (ft *FunctionType) String() string {
	var out bytes.Buffer
	out.WriteString(ft.Token.Kind)
	out.WriteString("(")

	for id, v := range ft.Args {
		out.WriteString(v.String())
		if id+1 <= len(ft.Args)-1 {
			out.WriteString(",")
		}
	}
	out.WriteString(") -> (")
	for id, v := range ft.Return {
		out.WriteString(v.String())
		if id+1 <= len(ft.Return)-1 {
			out.WriteString(",")
		}
	}

	out.WriteString(")")
	return out.String()
}

type PointerType struct {
	Token lexer.Token
	Kind  TypeKind
	Right Type
}

func (ptr *PointerType) Type() TypeKind {
	return ptr.Kind
}
func (ptr *PointerType) String() string {
	var out bytes.Buffer
	out.WriteString(ptr.Token.Kind)
	out.WriteString(ptr.Right.String())
	return out.String()
}

type Comment struct {
	Token lexer.Token // the token.LET token
	Value string
}

func (c *Comment) statementNode()        {}
func (c *Comment) TokenLiteral() string  { return c.Token.Text }
func (c *Comment) GetToken() lexer.Token { return c.Token }
func (c *Comment) String() string {
	return "/* " + c.Value + " */"
}

type Declaration struct {
	Token     lexer.Token               // the token.LET token
	Mutable   bool                      // indicates if the vars are mutable or not
	Inline    bool                      // for function inline
	Directive map[string]*StringLiteral // for non usual stuff such as #distinct types
	Type      Type
	Name      []*Identifier
	Value     []Expression
}

func (ls *Declaration) statementNode()        {}
func (ls *Declaration) TokenLiteral() string  { return ls.Token.Text }
func (nt *Declaration) GetToken() lexer.Token { return nt.Token }
func (ls *Declaration) String() string {
	var out bytes.Buffer
	out.WriteString(ls.TokenLiteral() + " ")
	for idx, name := range ls.Name {
		out.WriteString(name.String())
		if idx+1 <= len(ls.Name)-1 {
			out.WriteString(", ")
		}
	}
	if ls.Type != nil {
		out.WriteString(": " + ls.Type.String())
	}
	out.WriteString(" = ")
	for _, v := range ls.Value {
		out.WriteString(v.String())
	}
	return out.String()
}

type TypeStatement struct {
	Token    lexer.Token
	Alias    *Identifier
	Type     Type
	Distinct bool
}

func (ls *TypeStatement) statementNode()        {}
func (ls *TypeStatement) TokenLiteral() string  { return ls.Token.Text }
func (nt *TypeStatement) GetToken() lexer.Token { return nt.Token }
func (ls *TypeStatement) String() string {
	var out bytes.Buffer
	out.WriteString(ls.TokenLiteral() + " ")
	out.WriteString(ls.Alias.String() + " = ")
	out.WriteString(ls.Type.String())
	return out.String()
}

type DirectiveType int

const (
	_ DirectiveType = iota
	DistinctDirective
	InlineDirective
	MutUseDirective
	FallthroughDirective
	PartialDirective
	ForceDirective
)

type ImportStatement struct {
	Token      lexer.Token // the token.LET token
	Alias      *Identifier // alias for module name
	IntoScope  bool
	ModuleName *StringLiteral
}

func (ls *ImportStatement) statementNode()        {}
func (ls *ImportStatement) TokenLiteral() string  { return ls.Token.Text }
func (nt *ImportStatement) GetToken() lexer.Token { return nt.Token }
func (ls *ImportStatement) String() string {
	var out bytes.Buffer
	out.WriteString(ls.TokenLiteral() + " ")
	out.WriteString(ls.ModuleName.String())
	if ls.Alias != nil {
		out.WriteString("as " + ls.Alias.String())
	}
	return out.String()
}

type CastExpression struct {
	Token            lexer.Token
	ForceCast        bool // for #force directive
	TargetType       Type
	TargetExpression Expression
}

func (ce *CastExpression) expressionNode()       {}
func (ce *CastExpression) TokenLiteral() string  { return ce.Token.Text }
func (ce *CastExpression) GetToken() lexer.Token { return ce.Token }
func (ce *CastExpression) String() string {
	var out bytes.Buffer
	return out.String()
}

type Method struct {
	Key   *Identifier
	Value *FunctionExpression // any value type
}

type StructType struct {
	Token   lexer.Token // the token.LET token
	Fields  []*Declaration
	Methods []*Method
}

func (ss *StructType) Type() TypeKind {
	return TypeStruct
}
func (ss *StructType) TokenLiteral() string  { return ss.Token.Text }
func (nt *StructType) GetToken() lexer.Token { return nt.Token }
func (ss *StructType) String() string {
	var out bytes.Buffer
	out.WriteString(ss.TokenLiteral())
	out.WriteString(" { ")
	if ss.Fields != nil {
		for idx, field := range ss.Fields {
			out.WriteString(field.String())
			if idx <= len(ss.Fields)-1 {
				out.WriteString(", ")
			}
		}
	}

	if ss.Methods != nil {
		for idx, field := range ss.Methods {
			out.WriteString(field.Key.Value)
			out.WriteString(":")
			out.WriteString(field.Value.String())
			if idx+1 <= len(ss.Fields)-1 {
				out.WriteString(", ")
			}
		}
	}
	out.WriteString(" }")
	return out.String()
}

type EnumType struct {
	Token lexer.Token // the token.LET token
	Body  []*AssignExpression
}

func (ss *EnumType) Type() TypeKind {
	return TypeEnum
}
func (ss *EnumType) TokenLiteral() string  { return ss.Token.Text }
func (nt *EnumType) GetToken() lexer.Token { return nt.Token }
func (ss *EnumType) String() string {
	var out bytes.Buffer
	out.WriteString(ss.TokenLiteral())
	out.WriteString(" { ")
	if ss.Body != nil {
		for idx, expr := range ss.Body {
			out.WriteString(expr.String())
			if idx+1 <= len(ss.Body)-1 {
				out.WriteString(", ")
			}
		}
	}
	out.WriteString(" }")
	return out.String()
}

type ReturnStatement struct {
	Token        lexer.Token // the 'return' token
	ReturnValues []Expression
}

func (rs *ReturnStatement) statementNode()        {}
func (rs *ReturnStatement) TokenLiteral() string  { return rs.Token.Text }
func (nt *ReturnStatement) GetToken() lexer.Token { return nt.Token }
func (rs *ReturnStatement) String() string {
	var out bytes.Buffer
	out.WriteString(rs.TokenLiteral() + " ")
	for idx, retV := range rs.ReturnValues {
		out.WriteString(retV.String())
		if idx+1 <= len(rs.ReturnValues)-1 {
			out.WriteString(", ")
		}
	}
	return out.String()
}

type ExpressionStatement struct {
	Token      lexer.Token // the first token of the expression
	Expression Expression
}

func (es *ExpressionStatement) statementNode()        {}
func (es *ExpressionStatement) TokenLiteral() string  { return es.Token.Text }
func (nt *ExpressionStatement) GetToken() lexer.Token { return nt.Token }
func (es *ExpressionStatement) String() string {
	if es.Expression != nil {
		return es.Expression.String()
	}
	return ""
}

type WhileStatement struct {
	Token     lexer.Token
	Condition Expression
	Body      *BlockExpression
}

func (ws *WhileStatement) statementNode()        {}
func (ws *WhileStatement) TokenLiteral() string  { return ws.Token.Text }
func (nt *WhileStatement) GetToken() lexer.Token { return nt.Token }
func (ws *WhileStatement) String() string {
	var out bytes.Buffer
	out.WriteString("while ")
	out.WriteString(ws.Condition.String())
	out.WriteString(" { ")
	out.WriteString(ws.Body.String())
	out.WriteString(" }")
	return out.String()
}

type Pattern interface {
	pattern()
	String() string
}

type RangePattern struct {
	Token lexer.Token
	Op    string
	Start Expression
	End   Expression
}

func (fs *RangePattern) pattern()              {}
func (fs *RangePattern) expressionNode()       {}
func (fs *RangePattern) TokenLiteral() string  { return fs.Token.Text }
func (nt *RangePattern) GetToken() lexer.Token { return nt.Token }
func (fs *RangePattern) String() string {
	var out bytes.Buffer
	out.WriteString(fs.Start.String())
	out.WriteString(".." + fs.Op)
	out.WriteString(fs.End.String())
	return out.String()
}

type IterationPattern struct {
	Token     lexer.Token
	Start     Statement
	Condition Expression
	End       Expression
}

func (fs *IterationPattern) pattern()              {}
func (fs *IterationPattern) TokenLiteral() string  { return fs.Token.Text }
func (nt *IterationPattern) GetToken() lexer.Token { return nt.Token }
func (fs *IterationPattern) String() string {
	var out bytes.Buffer
	if fs.Start != nil {
		out.WriteString(fs.Start.String() + "; ")
	}
	if fs.Condition != nil {
		out.WriteString(fs.Condition.String())
	}
	if fs.End != nil {
		out.WriteString("; " + fs.End.String())
	}
	return out.String()
}

type IteratorIn struct {
	Token       lexer.Token
	Identifiers []*Identifier // mostly the variable
	Target      Expression    // target, either a map or an array
}

func (fs *IteratorIn) pattern()              {}
func (fs *IteratorIn) TokenLiteral() string  { return fs.Token.Text }
func (nt *IteratorIn) GetToken() lexer.Token { return nt.Token }
func (fs *IteratorIn) String() string {
	var out bytes.Buffer
	out.WriteString("for ")
	for idx, iden := range fs.Identifiers {
		out.WriteString(iden.String())
		if idx+1 <= len(fs.Identifiers)-1 {
			out.WriteString(", ")
		}
	}
	out.WriteString(" in ")
	out.WriteString(fs.Target.String())
	return out.String()
}

type ForStatement struct {
	Token   lexer.Token
	Pattern Pattern
	Body    *BlockExpression
}

func (fs *ForStatement) statementNode()        {}
func (fs *ForStatement) TokenLiteral() string  { return fs.Token.Text }
func (nt *ForStatement) GetToken() lexer.Token { return nt.Token }
func (fs *ForStatement) String() string {
	var out bytes.Buffer
	out.WriteString("for ")
	if fs.Pattern != nil {
		out.WriteString(fs.Pattern.String())
	}
	out.WriteString(" {\n")
	if fs.Body != nil {
		out.WriteString(fs.Body.String())
	}
	out.WriteString("}")
	return out.String()
}

type NextStatement struct {
	Token lexer.Token
}

func (fs *NextStatement) statementNode()        {}
func (fs *NextStatement) TokenLiteral() string  { return fs.Token.Text }
func (nt *NextStatement) GetToken() lexer.Token { return nt.Token }
func (fs *NextStatement) String() string        { return fs.TokenLiteral() }

type BreakStatement struct {
	Token lexer.Token
}

func (fs *BreakStatement) statementNode()        {}
func (fs *BreakStatement) TokenLiteral() string  { return fs.Token.Text }
func (nt *BreakStatement) GetToken() lexer.Token { return nt.Token }
func (fs *BreakStatement) String() string        { return fs.TokenLiteral() }

type Arg struct {
	Token        lexer.Token
	Name         *Identifier
	DefaultValue Expression
	Type         Type
}

type ReturnType struct {
	RtTypes []Type // support for multi return types
	MustUse bool   // refers to must_use directive
}

type FunctionExpression struct {
	Token  lexer.Token
	Self   *Identifier // this indicates the self key
	Args   []*Arg
	Return ReturnType
	Body   *BlockExpression
}

func (fn *FunctionExpression) expressionNode()       {}
func (fn *FunctionExpression) TokenLiteral() string  { return fn.Token.Text }
func (nt *FunctionExpression) GetToken() lexer.Token { return nt.Token }
func (fn *FunctionExpression) String() string {
	var out bytes.Buffer
	params := []string{}
	returnTypes := []string{}
	if fn.Self != nil {
		params = append(params, fn.Self.String())
	}
	for _, p := range fn.Args {
		formatParam := p.Name.String() + ":" + p.Type.String()
		params = append(params, formatParam)
	}
	for _, p := range fn.Return.RtTypes {
		returnTypes = append(returnTypes, p.String())
	}
	out.WriteString(fn.TokenLiteral())
	out.WriteString("(")
	out.WriteString(strings.Join(params, ", "))
	out.WriteString(") -> (")
	out.WriteString(strings.Join(returnTypes, ", "))
	out.WriteString(") {\n")
	out.WriteString(fn.Body.String())
	out.WriteString(" }")
	return out.String()
}

type ScopeStatement struct {
	Token lexer.Token
	Body  *BlockExpression
}

func (ss *ScopeStatement) statementNode()        {}
func (ss *ScopeStatement) TokenLiteral() string  { return ss.Token.Text }
func (nt *ScopeStatement) GetToken() lexer.Token { return nt.Token }
func (ss *ScopeStatement) String() string {
	var out bytes.Buffer
	out.WriteString("scope {\n")
	out.WriteString(ss.Body.String())
	out.WriteString("}")
	return out.String()
}

type Identifier struct {
	Token lexer.Token // the token.IDENT token
	Value string
}

func (i *Identifier) expressionNode()        {}
func (i *Identifier) TokenLiteral() string   { return i.Token.Text }
func (nt *Identifier) GetToken() lexer.Token { return nt.Token }
func (i *Identifier) String() string         { return i.Value }

type Literal interface {
	lit()
}

type IntegerLiteral struct {
	Token lexer.Token
	Value int64
}

func (il *IntegerLiteral) expressionNode()       {}
func (il *IntegerLiteral) lit()                  {}
func (il *IntegerLiteral) TokenLiteral() string  { return il.Token.Text }
func (nt *IntegerLiteral) GetToken() lexer.Token { return nt.Token }
func (il *IntegerLiteral) String() string        { return il.Token.Text }

type FloatLiteral struct {
	Token lexer.Token
	Value float64
}

func (fl *FloatLiteral) expressionNode()       {}
func (fl *FloatLiteral) lit()                  {}
func (fl *FloatLiteral) TokenLiteral() string  { return fl.Token.Text }
func (nt *FloatLiteral) GetToken() lexer.Token { return nt.Token }
func (fl *FloatLiteral) String() string        { return fl.Token.Text }

type StringLiteral struct {
	Token lexer.Token
	Value string
}

func (sl *StringLiteral) expressionNode()       {}
func (sl *StringLiteral) lit()                  {}
func (sl *StringLiteral) TokenLiteral() string  { return sl.Token.Text }
func (nt *StringLiteral) GetToken() lexer.Token { return nt.Token }
func (sl *StringLiteral) String() string {
	var out bytes.Buffer
	out.WriteString(`"`)
	out.WriteString(sl.Value)
	out.WriteString(`"`)
	return out.String()
}

type CharLiteral struct {
	Token lexer.Token
	Value rune
}

func (sl *CharLiteral) expressionNode()       {}
func (sl *CharLiteral) lit()                  {}
func (sl *CharLiteral) TokenLiteral() string  { return sl.Token.Text }
func (nt *CharLiteral) GetToken() lexer.Token { return nt.Token }
func (sl *CharLiteral) String() string {
	var out bytes.Buffer
	out.WriteString(`"`)
	out.WriteString(string(sl.Value))
	out.WriteString(`"`)
	return out.String()
}

type NulLiteral struct {
	Token lexer.Token
}

func (sl *NulLiteral) expressionNode()       {}
func (sl *NulLiteral) lit()                  {}
func (sl *NulLiteral) TokenLiteral() string  { return sl.Token.Text }
func (nt *NulLiteral) GetToken() lexer.Token { return nt.Token }
func (sl *NulLiteral) String() string {
	var out bytes.Buffer
	out.WriteString("nul")
	return out.String()
}

type BooleanLiteral struct {
	Token lexer.Token
	Value bool
}

func (bl *BooleanLiteral) expressionNode()       {}
func (bl *BooleanLiteral) lit()                  {}
func (bl *BooleanLiteral) TokenLiteral() string  { return bl.Token.Text }
func (nt *BooleanLiteral) GetToken() lexer.Token { return nt.Token }
func (bl *BooleanLiteral) String() string        { return bl.Token.Text }

type ArrayLiteral struct {
	Token    lexer.Token
	Type     Type
	Size     Expression // indicates the size of the array if it a fixed size array
	Elements []Expression
}

func (al *ArrayLiteral) expressionNode()       {}
func (il *ArrayLiteral) lit()                  {}
func (al *ArrayLiteral) TokenLiteral() string  { return al.Token.Text }
func (nt *ArrayLiteral) GetToken() lexer.Token { return nt.Token }
func (al *ArrayLiteral) String() string {
	var out bytes.Buffer
	out.WriteString("[")
	if al.Size != nil {
		out.WriteString(al.Size.String())
		out.WriteString("; ")
	}
	for idx, elem := range al.Elements {
		out.WriteString(elem.String())
		if idx+1 <= len(al.Elements)-1 {
			out.WriteString(",")
		}
	}
	out.WriteString("]")
	return out.String()
}

type MapLiteral struct {
	Token lexer.Token
	Type  Type
	Pairs map[Expression]Expression
}

func (ml *MapLiteral) expressionNode()       {}
func (il *MapLiteral) lit()                  {}
func (ml *MapLiteral) TokenLiteral() string  { return ml.Token.Text }
func (nt *MapLiteral) GetToken() lexer.Token { return nt.Token }
func (ml *MapLiteral) String() string {
	var out bytes.Buffer
	pairs := []string{}
	for key, value := range ml.Pairs {
		pairs = append(pairs, key.String()+": "+value.String())
	}
	out.WriteString("{")
	out.WriteString(strings.Join(pairs, ", "))
	out.WriteString("}")
	return out.String()
}

type UnaryExpression struct {
	Token    lexer.Token // the token.IDENT token
	Operator string
	Right    Expression
}

func (b *UnaryExpression) expressionNode()        {}
func (b *UnaryExpression) TokenLiteral() string   { return b.Token.Text }
func (nt *UnaryExpression) GetToken() lexer.Token { return nt.Token }
func (u *UnaryExpression) String() string {
	var out bytes.Buffer
	out.WriteString("(")
	out.WriteString(u.Operator)
	out.WriteString(u.Right.String())
	out.WriteString(")")
	return out.String()
}

type BinaryExpression struct {
	Token    lexer.Token // the token.IDENT token
	Operator string
	Left     Expression
	Right    Expression
}

func (b *BinaryExpression) expressionNode()        {}
func (b *BinaryExpression) TokenLiteral() string   { return b.Token.Text }
func (nt *BinaryExpression) GetToken() lexer.Token { return nt.Token }
func (b *BinaryExpression) String() string {
	var out bytes.Buffer
	out.WriteString("(")
	out.WriteString(b.Left.String())
	out.WriteString(" " + b.Operator + " ")
	out.WriteString(b.Right.String())
	out.WriteString(")")
	return out.String()
}

type AssignStatement struct {
	Token lexer.Token // the token.IDENT token
	Left  []*Identifier
	Right []Expression
}

func (b *AssignStatement) statementNode()         {}
func (b *AssignStatement) TokenLiteral() string   { return b.Token.Text }
func (nt *AssignStatement) GetToken() lexer.Token { return nt.Token }
func (b *AssignStatement) String() string {
	var out bytes.Buffer
	for _, elem := range b.Left {
		out.WriteString(elem.String())
	}
	out.WriteString(" = ")
	for _, elem := range b.Right {
		out.WriteString(elem.String())
	}
	return out.String()
}

type BlockExpression struct {
	Token lexer.Token
	Body  []Statement
}

func (bs *BlockExpression) expressionNode()       {}
func (bs *BlockExpression) TokenLiteral() string  { return bs.Token.Text }
func (nt *BlockExpression) GetToken() lexer.Token { return nt.Token }
func (bs *BlockExpression) String() string {
	var out bytes.Buffer
	for _, s := range bs.Body {
		out.WriteString("  " + s.String() + "\n")
	}
	return out.String()
}

type AssignExpression struct {
	Token lexer.Token // the token.IDENT token
	Left  []Expression
	Right []Expression
}

func (b *AssignExpression) expressionNode()        {}
func (b *AssignExpression) TokenLiteral() string   { return b.Token.Text }
func (nt *AssignExpression) GetToken() lexer.Token { return nt.Token }
func (b *AssignExpression) String() string {
	var out bytes.Buffer
	for _, elem := range b.Left {
		out.WriteString(elem.String())
	}
	out.WriteString(" = ")
	for _, elem := range b.Right {
		out.WriteString(elem.String())
	}
	return out.String()
}

type IfExpression struct {
	Token       lexer.Token
	Condition   Expression
	Consequence *BlockExpression
	Alternative Expression
}

func (ie *IfExpression) expressionNode()       {}
func (ie *IfExpression) TokenLiteral() string  { return ie.Token.Text }
func (nt *IfExpression) GetToken() lexer.Token { return nt.Token }
func (ie *IfExpression) String() string {
	var out bytes.Buffer
	out.WriteString("if ")
	out.WriteString(ie.Condition.String())
	out.WriteString(" {\n")
	out.WriteString(ie.Consequence.String())
	out.WriteString("}")
	if ie.Alternative != nil {
		out.WriteString(" else ")
		alternative, ok := ie.Alternative.(*IfExpression)
		if !ok {
			out.WriteString("{ ")
		}
		if ok {
			out.WriteString(alternative.String())
		} else {
			out.WriteString(ie.Alternative.String())
		}
		if !ok {
			out.WriteString(" }")
		}
	}
	return out.String()
}

type Case struct {
	Token      lexer.Token
	ArmPattern []Expression
	Body       *BlockExpression
	Break      bool
}

type SwitchExpression struct {
	Token        lexer.Token
	Condition    Expression
	Cases        []*Case
	PartialCheck bool // refers to #partial directive
}

func (ie *SwitchExpression) expressionNode()       {}
func (ie *SwitchExpression) TokenLiteral() string  { return ie.Token.Text }
func (nt *SwitchExpression) GetToken() lexer.Token { return nt.Token }
func (ie *SwitchExpression) String() string {
	var out bytes.Buffer
	out.WriteString("switch ")
	out.WriteString(ie.Condition.String())
	out.WriteString(" {\n")
	for i, cs := range ie.Cases {
		out.WriteString("case ")
		arms := []string{}
		for _, arm := range cs.ArmPattern {
			arms = append(arms, arm.String())
		}
		out.WriteString(strings.Join(arms, ", ") + ":\n")
		out.WriteString(cs.Body.String())
		if i+1 <= len(ie.Cases)-1 {
			out.WriteString("\n")
		}
	}
	out.WriteString("} ")
	return out.String()
}

type Param struct {
	Token lexer.Token
	Name  *Identifier
	Value Expression
}

type CallExpression struct {
	Token    lexer.Token // The '(' token
	Function *Identifier // Identifier
	Args     []Param
	Type     Type
}

func (ce *CallExpression) expressionNode()       {}
func (ce *CallExpression) TokenLiteral() string  { return ce.Token.Text }
func (nt *CallExpression) GetToken() lexer.Token { return nt.Token }
func (ce *CallExpression) String() string {
	var out bytes.Buffer
	args := []string{}
	for _, a := range ce.Args {
		args = append(args, a.Value.String())
	}
	out.WriteString(ce.Function.String())
	out.WriteString("(")
	out.WriteString(strings.Join(args, ", "))
	out.WriteString(")")
	return out.String()
}

type IndexExpression struct {
	Token lexer.Token // The [ token
	Left  Expression
	Range bool
	Start Expression
	End   Expression
}

func (ie *IndexExpression) expressionNode()       {}
func (ie *IndexExpression) TokenLiteral() string  { return ie.Token.Text }
func (nt *IndexExpression) GetToken() lexer.Token { return nt.Token }
func (ie *IndexExpression) String() string {
	var out bytes.Buffer
	out.WriteString(ie.Left.String())
	out.WriteString("[")
	if ie.Start != nil {
		out.WriteString(ie.Start.String())
	}
	if ie.Range {
		out.WriteString("..")
	}
	if ie.End != nil {
		out.WriteString(ie.End.String())
	}
	out.WriteString("]")
	return out.String()
}

type MemberShipExpression struct {
	Token    lexer.Token // The [ token
	Object   Expression
	Property Expression
}

func (me *MemberShipExpression) expressionNode()       {}
func (me *MemberShipExpression) TokenLiteral() string  { return me.Token.Text }
func (nt *MemberShipExpression) GetToken() lexer.Token { return nt.Token }
func (me *MemberShipExpression) String() string {
	var out bytes.Buffer
	out.WriteString(me.Object.String())
	out.WriteString(".")
	out.WriteString(me.Property.String())
	return out.String()
}

type FieldInstance struct {
	Key   *Identifier
	Value Expression // any value type
}

type StructInstanceExpression struct {
	Token lexer.Token // The [ token
	Left  Expression
	Body  []FieldInstance
}

func (sie *StructInstanceExpression) expressionNode()      {}
func (sie *StructInstanceExpression) TokenLiteral() string { return sie.Token.Text }
func (nt *StructInstanceExpression) GetToken() lexer.Token { return nt.Token }
func (sie *StructInstanceExpression) String() string {
	var out bytes.Buffer
	out.WriteString("(")
	out.WriteString(sie.Left.String())
	out.WriteString("[")
	if sie.Body != nil {
		for _, field := range sie.Body {
			out.WriteString(field.Key.Value)
			out.WriteString(":")
			out.WriteString(fmt.Sprintf("%v", field.Value))
		}
	}
	out.WriteString("])")
	return out.String()
}
