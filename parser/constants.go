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
		"skip":   TokenSkip,
		"true":   TokenBool,
		"false":  TokenBool,
	}

	binOperators = map[TokenKind]Operator{
		TokenEquals:         "==",
		TokenGreater:        ">",
		TokenGreaterOrEqual: ">=",
		TokenLess:           "<",
		TokenLessOrEqual:    "<=",
	}

	unaryOperators = map[TokenKind]Operator{
		TokenEquals: "!",
	}
)
