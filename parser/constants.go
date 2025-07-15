package parser

type Operator = string

var (
	keywords = map[string]TokenKind{
		"let":    TokenLet,
		"var":    TokenVar,
		"if":     TokenIf,
		"else":   TokenElse,
		"fn":     TokenFn,
		"for":    TokenFor,
		"while":  TokenWhile,
		"import": TokenImport,
		"export": TokenExport,
		"return": TokenReturn,
		"skip":   TokenSkip,
		"true":   TokenTrue,
		"false":  TokenFalse,
	}

	binOperators = map[TokenKind]Operator{
		TokenEquals:         "==",
		TokenGreater:        ">",
		TokenGreaterOrEqual: ">=",
		TokenLess:           "<",
		TokenLessOrEqual:    "<=",
		TokenNotEquals:      "!=",
		TokenMultiply:       "*",
		TokenSlash:          "/",
		TokenPlus:           "+",
		TokenMinus:          "-",
	}

	unaryOperators = map[TokenKind]Operator{
		TokenExclamation: "!",
		TokenMinus:       "-",
	}
)
