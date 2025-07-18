package parser

type Operator = string

var (
	keywords = map[string]TokenKind{
		"let":    TokenLet,
		"var":    TokenVar,
		"type":   TokenType,
		"struct": TokenStruct,
		"if":     TokenIf,
		"else":   TokenElse,
		"fn":     TokenFn,
		"for":    TokenFor,
		"while":  TokenWhile,
		"import": TokenImport,
		"return": TokenReturn,
		"skip":   TokenSkip,
		"array":  TokenArray,
		"map":    TokenMap,
		"true":   TokenTrue,
		"false":  TokenFalse,
	}

	atomicTypes = map[string]TYPE{
		"int":    IntType,
		"float":  FloatType,
		"string": StringType,
		"bool":   BoolType,
		"void":   VoidType,
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
		TokenModule:         "%",
		TokenPlus:           "+",
		TokenMinus:          "-",
		TokenAssign:         "=",
		TokenAnd:            "&&",
		TokenOr:             "||",
	}

	unaryOperators = map[TokenKind]Operator{
		TokenExclamation: "!",
		TokenMinus:       "-",
	}
)
