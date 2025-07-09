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
		"true":           TokenBool,
		"false":          TokenBool,
	}
)
