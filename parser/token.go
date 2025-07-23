package parser

type TokenKind = string

const (

	// Keywords
	TokenLet    TokenKind = "let"
	TokenVar    TokenKind = "var"
	TokenType   TokenKind = "type"
	TokenStruct TokenKind = "struct"
	TokenFn     TokenKind = "fn"
	TokenFor    TokenKind = "for"
	TokenIn     TokenKind = "in"
	TokenWhile  TokenKind = "while"
	TokenSkip   TokenKind = "skip"
	TokenIf     TokenKind = "if"
	TokenElse   TokenKind = "else"
	TokenReturn TokenKind = "return"
	TokenImport TokenKind = "import"

	// Units
	TokenCurlyBraceOpen  TokenKind = "{"
	TokenCurlyBraceClose TokenKind = "}"
	TokenBracketOpen     TokenKind = "["
	TokenBracketClose    TokenKind = "]"
	TokenBraceOpen       TokenKind = "("
	TokenBraceClose      TokenKind = ")"
	TokenQuote           TokenKind = `"`
	TokenSingleQuote     TokenKind = `'`
	TokenColon           TokenKind = ":"
	TokenComma           TokenKind = ","
	TokenDot             TokenKind = "."

	// Arithmetic Operators
	TokenMinus          TokenKind = "-"
	TokenPlus           TokenKind = "+"
	TokenMultiply       TokenKind = "*"
	TokenSlash          TokenKind = "/"
	TokenModule         TokenKind = "%"
	TokenAssign         TokenKind = "="
	TokenEquals         TokenKind = "=="
	TokenNotEquals      TokenKind = "!="
	TokenGreater        TokenKind = ">"
	TokenLess           TokenKind = "<"
	TokenGreaterOrEqual TokenKind = ">="
	TokenLessOrEqual    TokenKind = "<="
	// Logical Operators
	TokenAnd         TokenKind = "&&"
	TokenOr          TokenKind = "||"
	TokenExclamation TokenKind = "!"

	// Comment
	TokenComment TokenKind = "#"

	// Var Naming
	TokenIdentifier TokenKind = "identifier"

	// Var Types
	TokenString TokenKind = "string"
	TokenInt    TokenKind = "int"
	TokenFloat  TokenKind = "float"
	TokenBool   TokenKind = "bool"
	TokenArray  TokenKind = "array"
	TokenMap    TokenKind = "map"

	// Error
	TokenError TokenKind = "error"

	// EOF
	TokenEOF TokenKind = "end of file"
)

type LiteralToken struct {
	Text string
	Kind TokenKind
}

type Lexer struct {
	Content []rune
	// help mainly in error detection when having multi file execution
	FilePath string
	Row      int
	Col      int
	Cur      int
}
