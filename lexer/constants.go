package lexer

type Operator = string

var (
	Keywords = map[string]TokenKind{
		"let":    TokenLet,
		"const":  TokenConst,
		"struct": TokenStruct,
		"enum":   TokenEnum,
		"if":     TokenIf,
		"else":   TokenElse,
		"match":  TokenMatch,
		"fn":     TokenFn,
		"for":    TokenFor,
		"in":     TokenIn,
		"while":  TokenWhile,
		"import": TokenImport,
		"return": TokenReturn,
		"skip":   TokenSkip,
		"true":   TokenBool,
		"false":  TokenBool,
	}

	BinOperators = map[TokenKind]Operator{
		TokenEquals:         "==",
		TokenGreater:        ">",
		TokenGreaterOrEqual: ">=",
		TokenLess:           "<",
		TokenLessOrEqual:    "<=",
		TokenNotEquals:      "!=",
		TokenMultiply:       "*",
		TokenSlash:          "/",
		TokenModule:         "%",
		TokenPlus:           "+",
		TokenMinus:          "-",
		TokenAssign:         "=",
		TokenAnd:            "&&",
		TokenOr:             "||",
	}

	UnaryOperators = map[TokenKind]Operator{
		TokenExclamation: "!",
		TokenMinus:       "-",
	}
)
