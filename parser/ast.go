package parser

import (
	"bytes"
	"fmt"
	"strings"
)

type TYPE = string

const (
	IntType    TYPE = "int"
	FloatType  TYPE = "float"
	StringType TYPE = "string"
	BoolType   TYPE = "bool"
	VoidType   TYPE = "void"
)

type Node interface {
	TokenLiteral() string
	String() string
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
	for _, s := range p.Statements {
		out.WriteString(s.String())
	}
	return out.String()
}

type NodeType struct {
	Token     Token
	Type      TYPE
	ChildType *NodeType
	Size      string
}

func (nt *NodeType) expressionNode()      {}
func (nt *NodeType) TokenLiteral() string { return nt.Token.Text }
func (nt *NodeType) String() string       {
	var out bytes.Buffer

	if len(nt.Size) > 0 {
		out.WriteString("[")
		out.WriteString(nt.Size)
		out.WriteString("]")
		out.WriteString(nt.ChildType.String())

		return out.String()
	}
	if nt.ChildType == nil {
		out.WriteString(nt.Type)
	} else {
		out.WriteString(nt.Type)
		out.WriteString("(")
		out.WriteString(nt.ChildType.String())
		out.WriteString(")")
	}
	return out.String()
}

type MapType struct {
	Token Token
	Type  TYPE
	Left  Expression
	Right Expression
}

func (mt *MapType) expressionNode()      {}
func (mt *MapType) TokenLiteral() string { return mt.Token.Text }
func (mt *MapType) String() string       {
	var out bytes.Buffer
	out.WriteString(mt.Type)
	out.WriteString("(")
	if mt.Left != nil {
		out.WriteString(mt.Left.String())
	}
	if mt.Right != nil {
	  out.WriteString(", ")
		out.WriteString(mt.Right.String())
	}
	out.WriteString(")")
	return out.String()
}

type LetStatement struct {
	Token        Token // the token.LET token
	Name         *Identifier
	ExplicitType Expression
	Value        Expression
}

func (ls *LetStatement) statementNode()       {}
func (ls *LetStatement) TokenLiteral() string { return ls.Token.Text }

func (ls *LetStatement) String() string {
	var out bytes.Buffer
	out.WriteString(ls.TokenLiteral() + " ")
	out.WriteString(ls.Name.String())
	out.WriteString(" = ")
	if ls.Value != nil {
		out.WriteString(ls.Value.String())
	}
	return out.String()
}

type TypeStatement struct {
	Token Token // the token.LET token
	Name  *Identifier
	Value Expression
}

func (ts *TypeStatement) statementNode()       {}
func (ts *TypeStatement) TokenLiteral() string { return ts.Token.Text }

func (ts *TypeStatement) String() string {
	var out bytes.Buffer
	out.WriteString(ts.TokenLiteral() + " ")
	out.WriteString(ts.Name.String())
	out.WriteString(" = ")
	if ts.Value != nil {
		out.WriteString(ts.Value.String())
	}
	return out.String()
}

type Field struct {
	Key   *Identifier
	Value Expression // any value type
}

type StructStatement struct {
	Token Token // the token.LET token
	Name  *Identifier
	Body  []Field
}

func (ss *StructStatement) statementNode()       {}
func (ss *StructStatement) TokenLiteral() string { return ss.Token.Text }

func (ss *StructStatement) String() string {
	var out bytes.Buffer
	out.WriteString(ss.TokenLiteral() + " ")
	out.WriteString(ss.Name.String())
	out.WriteString(" { ")
	if ss.Body != nil {
		for idx , field := range ss.Body {
			out.WriteString(field.Key.Value)
			out.WriteString(":")
			out.WriteString(fmt.Sprintf("%v", field.Value))
			if idx + 1 <= len(ss.Body) - 1 {
				out.WriteString(", ")
			}
		}
	}
	out.WriteString(" }")
	return out.String()
}

type ReturnStatement struct {
	Token       Token // the 'return' token
	ReturnValue Expression
}

func (rs *ReturnStatement) statementNode()       {}
func (rs *ReturnStatement) TokenLiteral() string { return rs.Token.Text }
func (rs *ReturnStatement) String() string {
	var out bytes.Buffer
	out.WriteString(rs.TokenLiteral() + " ")
	if rs.ReturnValue != nil {
		out.WriteString(rs.ReturnValue.String())
	}
	return out.String()
}

type ExpressionStatement struct {
	Token      Token // the first token of the expression
	Expression Expression
}

func (es *ExpressionStatement) statementNode()       {}
func (es *ExpressionStatement) TokenLiteral() string { return es.Token.Text }

func (es *ExpressionStatement) String() string {
	if es.Expression != nil {
		return es.Expression.String()
	}
	return ""
}

type WhileStatement struct {
	Token     Token
	Condition Expression
	Body      *BlockStatement
}

func (ws *WhileStatement) statementNode()       {}
func (ws *WhileStatement) TokenLiteral() string { return ws.Token.Text }

func (ws *WhileStatement) String() string {
	var out bytes.Buffer
	out.WriteString("while ")
	out.WriteString(ws.Condition.String())
	out.WriteString(" { ")
	out.WriteString(ws.Body.String())
	out.WriteString(" }")
	return out.String()
}

type Identifier struct {
	Token Token // the token.IDENT token
	Value string
}

func (i *Identifier) expressionNode()      {}
func (i *Identifier) TokenLiteral() string { return i.Token.Text }

func (i *Identifier) String() string { return i.Value }

type IntegerLiteral struct {
	Token Token
	Value int64
}

func (il *IntegerLiteral) expressionNode()      {}
func (il *IntegerLiteral) TokenLiteral() string { return il.Token.Text }
func (il *IntegerLiteral) String() string       { return il.Token.Text }

type FloatLiteral struct {
	Token Token
	Value float64
}

func (fl *FloatLiteral) expressionNode()      {}
func (fl *FloatLiteral) TokenLiteral() string { return fl.Token.Text }
func (fl *FloatLiteral) String() string       { return fl.Token.Text }

type StringLiteral struct {
	Token Token
	Value string
}

func (sl *StringLiteral) expressionNode()      {}
func (sl *StringLiteral) TokenLiteral() string { return sl.Token.Text }
func (sl *StringLiteral) String() string       {
	var out bytes.Buffer
	out.WriteString(`"`)
	out.WriteString(sl.Value)
	out.WriteString(`"`)
	return out.String()
}

type BooleanLiteral struct {
	Token Token
	Value bool
}

func (bl *BooleanLiteral) expressionNode()      {}
func (bl *BooleanLiteral) TokenLiteral() string { return bl.Token.Text }
func (bl *BooleanLiteral) String() string       { return bl.Token.Text }

type ArrayLiteral struct {
	Token    Token
	Elements []Expression
}

func (al *ArrayLiteral) expressionNode()      {}
func (al *ArrayLiteral) TokenLiteral() string { return al.Token.Text }
func (al *ArrayLiteral) String() string       {
	var out bytes.Buffer
	out.WriteString("[")
	for idx , elem := range al.Elements {
		out.WriteString(elem.String())
		if idx + 1 <= len(al.Elements) - 1 {
			out.WriteString(", ")
		}
	}
	out.WriteString("]")
	return out.String()
}

type MapLiteral struct {
	Token Token
	Pairs map[Expression]Expression
}

func (ml *MapLiteral) expressionNode()      {}
func (ml *MapLiteral) TokenLiteral() string { return ml.Token.Text }
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
	Token    Token // the token.IDENT token
	Operator string
	Right    Expression
}

func (b *UnaryExpression) expressionNode()      {}
func (b *UnaryExpression) TokenLiteral() string { return b.Token.Text }
func (u *UnaryExpression) String() string {
	var out bytes.Buffer
	out.WriteString("(")
	out.WriteString(u.Operator)
	out.WriteString(u.Right.String())
	out.WriteString(")")
	return out.String()
}

type BinaryExpression struct {
	Token    Token // the token.IDENT token
	Operator string
	Left     Expression
	Right    Expression
}

func (b *BinaryExpression) expressionNode()      {}
func (b *BinaryExpression) TokenLiteral() string { return b.Token.Text }
func (b *BinaryExpression) String() string {
	var out bytes.Buffer
	out.WriteString("(")
	out.WriteString(b.Left.String())
	out.WriteString(" " + b.Operator + " ")
	out.WriteString(b.Right.String())
	out.WriteString(")")
	return out.String()
}

type BlockStatement struct {
	Token Token
	Body  []Statement
}

func (bs *BlockStatement) expressionNode()      {}
func (bs *BlockStatement) TokenLiteral() string { return bs.Token.Text }
func (bs *BlockStatement) String() string {
	var out bytes.Buffer
	for _, s := range bs.Body {
		out.WriteString(s.String())
	}
	return out.String()
}

type IfExpression struct {
	Token       Token
	Condition   Expression
	Consequence *BlockStatement
	Alternative Expression
}

func (ie *IfExpression) expressionNode()      {}
func (ie *IfExpression) TokenLiteral() string { return ie.Token.Text }
func (ie *IfExpression) String() string {
	var out bytes.Buffer
	out.WriteString("if ")
	out.WriteString(ie.Condition.String())
	out.WriteString(" { ")
	out.WriteString(ie.Consequence.String())
	out.WriteString(" }")
	if ie.Alternative != nil {
		out.WriteString(" else ")
		alternative , ok := ie.Alternative.(*IfExpression)
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

type FunctionStatement struct {
	Token      Token
	Name       string
	Args       []*Identifier
	ReturnType Expression
	Body       *BlockStatement
}

func (fn *FunctionStatement) statementNode()       {}
func (fn *FunctionStatement) TokenLiteral() string { return fn.Token.Text }
func (fn *FunctionStatement) String() string {
	var out bytes.Buffer
	params := []string{}
	for _, p := range fn.Args {
		params = append(params, p.String())
	}
	out.WriteString(fn.TokenLiteral())
	out.WriteString("(")
	out.WriteString(strings.Join(params, ", "))
	out.WriteString(") ")
	out.WriteString(fn.Body.String())
	return out.String()
}

type CallExpression struct {
	Token    Token      // The '(' token
	Function Identifier // Identifier
	Args     []Expression
}

func (ce *CallExpression) expressionNode()      {}
func (ce *CallExpression) TokenLiteral() string { return ce.Token.Text }
func (ce *CallExpression) String() string {
	var out bytes.Buffer
	args := []string{}
	for _, a := range ce.Args {
		args = append(args, a.String())
	}
	out.WriteString(ce.Function.String())
	out.WriteString("(")
	out.WriteString(strings.Join(args, ", "))
	out.WriteString(")")
	return out.String()
}

type IndexExpression struct {
	Token Token // The [ token
	Left  Expression
	Index Expression
}

func (ie *IndexExpression) expressionNode()      {}
func (ie *IndexExpression) TokenLiteral() string { return ie.Token.Text }
func (ie *IndexExpression) String() string {
	var out bytes.Buffer
	out.WriteString("(")
	out.WriteString(ie.Left.String())
	out.WriteString("[")
	out.WriteString(ie.Index.String())
	out.WriteString("])")
	return out.String()
}

type MemberShipExpression struct {
	Token    Token // The [ token
	Object   Expression
	Property Expression
}

func (me *MemberShipExpression) expressionNode()      {}
func (me *MemberShipExpression) TokenLiteral() string { return me.Token.Text }
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
	Token Token // The [ token
	Left  Expression
	Body  []FieldInstance
}

func (sie *StructInstanceExpression) expressionNode()      {}
func (sie *StructInstanceExpression) TokenLiteral() string { return sie.Token.Text }
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

type Parser struct {
	Tokens         []Token
	FilePath       string
	Errors         []error
	Pos            int
	prefixParseFns map[TokenKind]prefixParseFn
	infixParseFns  map[TokenKind]infixParseFn
	internalFlags  []string
}
