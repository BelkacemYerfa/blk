package parser

import (
	"bytes"
	"strings"
)

type TYPE = string

const (
	IntType    TYPE = "int"
	FloatType  TYPE = "float"
	StringType TYPE = "string"
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

type LetStatement struct {
	Token        Token // the token.LET token
	Name         *Identifier
	ExplicitType TYPE
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
	out.WriteString(";")
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
	out.WriteString(";")
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
func (sl *StringLiteral) String() string       { return sl.Token.Text }

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
func (al *ArrayLiteral) String() string       { return al.Token.Text }

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

func (bk *BlockStatement) expressionNode()      {}
func (bk *BlockStatement) TokenLiteral() string { return bk.Token.Text }
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
	Alternative *BlockStatement
}

func (ie *IfExpression) expressionNode()      {}
func (ie *IfExpression) TokenLiteral() string { return ie.Token.Text }
func (ie *IfExpression) String() string {
	var out bytes.Buffer
	out.WriteString("if")
	out.WriteString(ie.Condition.String())
	out.WriteString(" ")
	out.WriteString(ie.Consequence.String())
	if ie.Alternative != nil {
		out.WriteString("else ")
		out.WriteString(ie.Alternative.String())
	}
	return out.String()
}

type FnExpression struct {
	Token      Token
	Name       string
	Args       []*Identifier
	ReturnType TYPE
	Body       *BlockStatement
}

func (fn *FnExpression) expressionNode()      {}
func (fn *FnExpression) TokenLiteral() string { return fn.Token.Text }
func (fn *FnExpression) String() string {
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

type Parser struct {
	Tokens         []Token
	FilePath       string
	Errors         []error
	Pos            int
	prefixParseFns map[TokenKind]prefixParseFn
	infixParseFns  map[TokenKind]infixParseFn
}
