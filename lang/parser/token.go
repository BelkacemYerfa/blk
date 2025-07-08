package parser

type TokenKind = string

const (

	// Commands (Reserved Keywords)
	TokenPush          TokenKind = "push"
	TokenTrim          TokenKind = "trim"
	TokenExport        TokenKind = "export"
	TokenThumbnailFrom TokenKind = "thumbnail_from"
	TokenConcat        TokenKind = "concat"
	TokenSet           TokenKind = "set"
	TokenUse           TokenKind = "use"
	TokenOn            TokenKind = "on"
	TokenProcess       TokenKind = "process"

	// Block Units
	TokenCurlyBraceOpen  TokenKind = "{"
	TokenCurlyBraceClose TokenKind = "}"
	TokenQuote           TokenKind = `"`
	TokenColon           TokenKind = ":"
	TokenMinus           TokenKind = "-"
	TokenPlus            TokenKind = "+"

	// Comment
	TokenComment TokenKind = "#"

	// Var Naming
	TokenIdentifier TokenKind = "identifier"

	// Var Types
	TokenString TokenKind = "string"
	TokenTime   TokenKind = "time"
	TokenNumber TokenKind = "number"
	TokenBool   TokenKind = "bool"

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
