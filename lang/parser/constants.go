package parser

var (
	keywords = map[string]TokenKind{
		"set":            TokenSet,
		"push":           TokenPush,
		"use":            TokenUse,
		"on":             TokenOn,
		"export":         TokenExport,
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
