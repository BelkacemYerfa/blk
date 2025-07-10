package parser

type (
	Statement  = string
	Expression = string
	Type       = string
	Operator   = string
)

const (
	// Statements
	PushStatement      Statement = "PushStatement"
	TrimStatement      Statement = "TrimStatement"
	ExportStatement    Statement = "ExportStatement"
	ConcatStatement    Statement = "ConcatStatement"
	ThumbnailStatement Statement = "ThumbnailStatement"
	SetStatement       Statement = "SetStatement"
	UseStatement       Statement = "UseStatement"
	ProcessStatement   Statement = "ProcessStatement"
	IfStatement        Statement = "IfStatement"
	ElseStatement      Statement = "ElseStatement"
	ForeachStatement   Statement = "ForeachStatement"
	SkipStatement      Statement = "SkipStatement"

	// Expressions
	LiteralExpression    Expression = "LiteralExpression"
	IdentifierExpression Expression = "IdentifierExpression"
	ObjectExpression     Expression = "ObjectExpression"
	MemberExpression     Expression = "MemberAccessExpression"
	BinaryExpression     Expression = "BinaryExpression"
	UnaryExpression      Expression = "UnaryExpression"

	// Operators
	EqualsOperator         Operator = "=="
	GreaterOperator        Operator = ">"
	GreaterOrEqualOperator Operator = ">="
	LessOperator           Operator = "<"
	LessOrEqualOperator    Operator = "<="
	ExclamationOperator    Operator = "!"

	// Types
	// Primitive
	NumberType  Type = "NumberType"
	BooleanType Type = "BooleanType"
	StringType  Type = "StringType"
	// Custom
	IdentifierType Type = "IdentifierType"
	TimeType       Type = "TimeType"
	// Complex
	ObjectType Type = "ObjectType"
)

type Position struct {
	Row int
	Col int
}

type MemberAccessExpression struct {
	Name     string
	Property *MemberAccessExpression
}

type BinaryExpressionNode struct {
	Type     Expression
	Left     ExpressionNode
	Right    ExpressionNode
	Operator Operator
}

type UnaryExpressionNode struct {
	Type     Expression
	Operator Operator
	Right    ExpressionNode
}

type ExpressionNode struct {
	Type       Expression // "literal_expression", "identifier_expression", etc.
	Identifier string     // used when declaring variables (it will hold the value name)
	Value      any        // string, float64, bool, or even ObjectLiteral for identifier type
	ExprType   Type       // For type-checking: "number", "bool", etc.
	Position   Position
}

type StatementNode struct {
	Type     Statement       // e.g., "push", "set", etc.
	Params   []any           // can take an expression node or a binary expression node
	Body     []StatementNode // Only for process/batch/etc.
	Position Position
	Order    int
}

// we use this when an expression is an object expression
type ObjectLiteral map[string]ExpressionNode

type AST = []StatementNode

type Parser struct {
	Tokens []Token
	Pos    int
}
