package lexer

type TokenKind = string

const (

	// Keywords
	TokenLet    TokenKind = "let"
	TokenConst  TokenKind = "const"
	TokenStruct TokenKind = "struct"
	TokenSelf   TokenKind = "self"
	TokenEnum   TokenKind = "enum"
	TokenFn     TokenKind = "fn"
	TokenFor    TokenKind = "for"
	TokenIn     TokenKind = "in"
	TokenWhile  TokenKind = "while"
	TokenSkip   TokenKind = "skip"
	TokenBreak  TokenKind = "break"
	TokenUse    TokenKind = "use"
	TokenIf     TokenKind = "if"
	TokenElse   TokenKind = "else"
	TokenMatch  TokenKind = "match"
	TokenReturn TokenKind = "return"
	TokenImport TokenKind = "import"
	TokenAs     TokenKind = "as"

	// nul values
	TokenNul TokenKind = "nul"

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
	TokenRange           TokenKind = ".."
	TokenQuestion        TokenKind = "?"

	// Arithmetic Operators
	TokenMinus          TokenKind = "-"
	TokenPlus           TokenKind = "+"
	TokenMultiply       TokenKind = "*"
	TokenSlash          TokenKind = "/"
	TokenModule         TokenKind = "%"
	TokenEquals         TokenKind = "=="
	TokenNotEquals      TokenKind = "!="
	TokenGreater        TokenKind = ">"
	TokenLess           TokenKind = "<"
	TokenGreaterOrEqual TokenKind = ">="
	TokenLessOrEqual    TokenKind = "<="
	TokenAssignMinus    TokenKind = "-="
	TokenAssignMinusOne TokenKind = "--"
	TokenAssignPlus     TokenKind = "+="
	TokenAssignPlusOne  TokenKind = "++"
	TokenAssignMultiply TokenKind = "*="
	TokenAssignSlash    TokenKind = "/="
	TokenAssignModule   TokenKind = "%="

	// Bind Operators
	TokenAssign TokenKind = "="
	TokenBind   TokenKind = "::"
	TokenWalrus TokenKind = ":="

	// Logical Operators
	TokenAnd         TokenKind = "&&"
	TokenOr          TokenKind = "||"
	TokenAssignAnd   TokenKind = "&&="
	TokenAssignOr    TokenKind = "||="
	TokenExclamation TokenKind = "!"

	// Comment
	TokenComment TokenKind = "#"

	// Var Naming
	TokenIdentifier TokenKind = "identifier"

	// Var Types
	TokenString TokenKind = "string"
	TokenChar   TokenKind = "char" // represents a rune
	TokenInt    TokenKind = "int"
	TokenFloat  TokenKind = "float"
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
