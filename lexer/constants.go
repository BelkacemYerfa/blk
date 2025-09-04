package lexer

type Operator = string

var (
	Keywords = map[string]TokenKind{
		"let":    TokenLet,
		"const":  TokenConst,
		"struct": TokenStruct,
		"self":   TokenSelf,
		"enum":   TokenEnum,
		"if":     TokenIf,
		"else":   TokenElse,
		"use":    TokenUse,
		"match":  TokenMatch,
		"fn":     TokenFn,
		"for":    TokenFor,
		"in":     TokenIn,
		"while":  TokenWhile,
		"import": TokenImport,
		"as":     TokenAs,
		"return": TokenReturn,
		"next":   TokenNext,
		"break":  TokenBreak,
		"true":   TokenBool,
		"false":  TokenBool,
		"nul":    TokenNul,
	}

	BinOperators = map[TokenKind]Operator{
		TokenEquals:              "==",
		TokenGreater:             ">",
		TokenGreaterOrEqual:      ">=",
		TokenLess:                "<",
		TokenLessOrEqual:         "<=",
		TokenNotEquals:           "!=",
		TokenMultiply:            "*",
		TokenSlash:               "/",
		TokenModule:              "%",
		TokenPlus:                "+",
		TokenMinus:               "-",
		TokenAssignMinus:         "-=",
		TokenAssignMinusOne:      "--",
		TokenAssignPlus:          "+=",
		TokenAssignPlusOne:       "++",
		TokenAssignModule:        "%=",
		TokenAssignMultiply:      "*=",
		TokenAssignSlash:         "/=",
		TokenAnd:                 "&&",
		TokenOr:                  "||",
		TokenAssignAnd:           "&&=",
		TokenAssignOr:            "||=",
		TokenBitAnd:              "&",
		TokenBitOr:               "|",
		TokenBitXOR:              "^",
		TokenBitRightShift:       ">>",
		TokenBitLeftShift:        "<<",
		TokenAssignBitAnd:        "&=",
		TokenAssignBitOr:         "|=",
		TokenAssignBitXor:        "^=",
		TokenAssignBitRightShift: ">>=",
		TokenAssignBitLeftShift:  "<<=",
	}

	UnaryOperators = map[TokenKind]Operator{
		TokenExclamation: "!",
		TokenMinus:       "-",
		TokenBitNot:      "~",
	}
)
