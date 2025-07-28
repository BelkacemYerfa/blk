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
		"scope":  TokenScope,
		"fn":     TokenFn,
		"for":    TokenFor,
		"in":     TokenIn,
		"while":  TokenWhile,
		"import": TokenImport,
		"return": TokenReturn,
		"skip":   TokenSkip,
		"array":  TokenArray,
		"map":    TokenMap,
		"true":   TokenBool,
		"false":  TokenBool,
	}

	AtomicTypes = map[string]TYPE{
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
