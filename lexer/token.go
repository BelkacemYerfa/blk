package lexer

type TokenKind = string

const (

	// Keywords
	TokenLet    TokenKind = "let"
	TokenConst  TokenKind = "const"
	TokenSelf   TokenKind = "self"
	TokenFn     TokenKind = "fn"
	TokenFor    TokenKind = "for"
	TokenIn     TokenKind = "in"
	TokenNext   TokenKind = "next"
	TokenBreak  TokenKind = "break"
	TokenDo     TokenKind = "do"
	TokenSwitch TokenKind = "switch"
	TokenCase   TokenKind = "case"
	TokenIf     TokenKind = "if"
	TokenElse   TokenKind = "else"
	TokenReturn TokenKind = "return"
	TokenImport TokenKind = "import"
	TokenCast   TokenKind = "cast"
	TokenType   TokenKind = "type"
	TokenTest   TokenKind = "test"

	// nul value
	TokenNul TokenKind = "nul"

	// Directives
	TokenDistinct    TokenKind = "#distinct"
	TokenInline      TokenKind = "#inline"
	TokenNoInline    TokenKind = "#no_inline"
	TokenMustUse     TokenKind = "#must_use"
	TokenFallthrough TokenKind = "#fallthrough"
	TokenPartial     TokenKind = "#partial"
	TokenForce       TokenKind = "#force"
	TokenBake        TokenKind = "#bake"
	TokenDeprecated  TokenKind = "#deprecated"
	TokenScope       TokenKind = "#scope"
	TokenNoInit      TokenKind = "#no_init"
	TokenLine        TokenKind = "#line"
	TokenDir         TokenKind = "#dir"
	TokenFile        TokenKind = "#file"
	TokenInit        TokenKind = "#init"
	TokenFini        TokenKind = "#fini"

	// Units
	TokenCurlyBraceOpen  TokenKind = "{"
	TokenCurlyBraceClose TokenKind = "}"
	TokenBracketOpen     TokenKind = "["
	TokenBracketClose    TokenKind = "]"
	TokenBraceOpen       TokenKind = "("
	TokenBraceClose      TokenKind = ")"
	TokenQuote           TokenKind = `"`
	TokenSingleQuote     TokenKind = `'`
	TokenRawString       TokenKind = "`"
	TokenColon           TokenKind = ":"
	TokenComma           TokenKind = ","
	TokenSemiColon       TokenKind = ";"
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

	// Bitwise Operators
	TokenBitAnd              TokenKind = "&"
	TokenBitOr               TokenKind = "|"
	TokenBitNot              TokenKind = "~"
	TokenBitXOR              TokenKind = "^"
	TokenBitRightShift       TokenKind = ">>"
	TokenBitLeftShift        TokenKind = "<<"
	TokenAssignBitAnd        TokenKind = "&="
	TokenAssignBitOr         TokenKind = "|="
	TokenAssignBitXor        TokenKind = "^="
	TokenAssignBitRightShift TokenKind = ">>="
	TokenAssignBitLeftShift  TokenKind = "<<="

	// Bind Operators
	TokenAssign TokenKind = "="

	// Logical Operators
	TokenAnd         TokenKind = "&&"
	TokenOr          TokenKind = "||"
	TokenAssignAnd   TokenKind = "&&="
	TokenAssignOr    TokenKind = "||="
	TokenExclamation TokenKind = "!"

	// Comment
	TokenComment   TokenKind = "//"
	TokenDirective TokenKind = "#"

	// Arrow
	TokenArrow TokenKind = "->"

	// Var Naming
	TokenIdentifier TokenKind = "identifier"

	// Var Types
	TokenString  TokenKind = "string"
	TokenChar    TokenKind = "char"
	TokenBool    TokenKind = "bool"
	TokenSInt    TokenKind = "sint"
	TokenSInt8   TokenKind = "s8"
	TokenSInt16  TokenKind = "s16"
	TokenSInt32  TokenKind = "s32"
	TokenSInt64  TokenKind = "s64"
	TokenUInt    TokenKind = "uint"
	TokenUInt8   TokenKind = "u8"
	TokenUInt16  TokenKind = "u16"
	TokenUInt32  TokenKind = "u32"
	TokenUInt64  TokenKind = "u64"
	TokenFloat32 TokenKind = "f32"
	TokenFloat64 TokenKind = "f64"
	TokenMap     TokenKind = "map"
	TokenAny     TokenKind = "any"

	// Record types
	TokenStruct TokenKind = "struct"
	TokenEnum   TokenKind = "enum"
	TokenUnion  TokenKind = "union"

	// Types (used in the lexing phase)
	TokenStr               = "str"
	TokenInteger TokenKind = "integer"
	TokenFloat   TokenKind = "float"

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
