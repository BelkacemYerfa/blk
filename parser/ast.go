package parser

type Node interface {
	TokenLiteral() string
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

type LetStatement struct {
	Token Token // the token.LET token
	Name  *Identifier
	Value Expression
}

func (ls *LetStatement) statementNode()       {}
func (ls *LetStatement) TokenLiteral() string { return ls.Token.Kind }

type Identifier struct {
	Token Token // the token.IDENT token
	Value string
}

func (i *Identifier) expressionNode()      {}
func (i *Identifier) TokenLiteral() string { return i.Token.Kind }

type LiteralExpression struct {
	Token Token // the token.IDENT token
	Value any
}

func (l *LiteralExpression) expressionNode()      {}
func (l *LiteralExpression) TokenLiteral() string { return l.Token.Kind }

type BinaryExpression struct {
	Token    Token // the token.IDENT token
	Operator string
	Left     Expression
	Right    Expression
}

func (b *BinaryExpression) expressionNode()      {}
func (b *BinaryExpression) TokenLiteral() string { return b.Token.Kind }

type Parser struct {
	Tokens   []Token
	FilePath string
	Pos      int
}
