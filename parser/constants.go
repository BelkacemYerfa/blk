package parser

var (
	keywords = map[string]TokenKind{
		"set":            TokenSet,
		"push":           TokenPush,
		"export":         TokenExport,
		"pop":            TokenPop,
		"as":             TokenAs,
		"swap":           TokenSwap,
		"dup":            TokenDup,
		"clear":          TokenClear,
		"rotate":         TokenRotate,
		"print_stack":    TokenPrintStack,
		"use":            TokenUse,
		"on":             TokenOn,
		"trim":           TokenTrim,
		"thumbnail_from": TokenThumbnailFrom,
		"concat":         TokenConcat,
		"process":        TokenProcess,
		"if":             TokenIf,
		"else":           TokenElse,
		"foreach":        TokenForEach,
		"in":             TokenIn,
		"recurse":        TokenRecurse,
		"skip":           TokenSkip,
		"true":           TokenBool,
		"false":          TokenBool,
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
