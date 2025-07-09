package parser

type (
	Statement  = string
	Expression = string
	Type       = string
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

type ExpressionNode struct {
	Type     Expression // "literal_expression", "identifier_expression", etc.
	Value    any        // string, float64, bool, or even ObjectLiteral
	ExprType Type       // For type-checking: "number", "bool", etc.
	Position Position
}

type StatementNode struct {
	Type     Statement // e.g., "push", "set", etc.
	Params   []ExpressionNode
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
