package lexer

type Operator = string

var (
	Keywords = map[string]TokenKind{
		"let":    TokenLet,
		"const":  TokenConst,
		"self":   TokenSelf,
		"enum":   TokenEnum,
		"switch": TokenSwitch,
		"case":   TokenCase,
		"if":     TokenIf,
		"else":   TokenElse,
		"use":    TokenDo,
		"fn":     TokenFn,
		"for":    TokenFor,
		"in":     TokenIn,
		"import": TokenImport,
		"using":  TokenUsing,
		"return": TokenReturn,
		"next":   TokenNext,
		"break":  TokenBreak,
		"cast":   TokenCast,
		"true":   TokenBool,
		"false":  TokenBool,
		"nul":    TokenNul,
	}

	TypeKeywords = map[string]TokenKind{
		"string": TokenString,
		"char":   TokenChar,
		"bool":   TokenBool,
		"i8":     TokenInt8,
		"i16":    TokenInt16,
		"i32":    TokenInt32,
		"i64":    TokenInt64,
		"u8":     TokenUInt8,
		"u16":    TokenUInt16,
		"u32":    TokenUInt32,
		"u64":    TokenUInt64,
		"f32":    TokenFloat32,
		"f64":    TokenFloat64,
		"any":    TokenAny,
		"map":    TokenMap,
		"struct": TokenStruct,
	}

	Directives = map[string]TokenKind{
		"#distinct":    TokenDistinct,
		"#inline":      TokenInline,
		"#must_use":    TokenMustUse,
		"#fallthrough": TokenFallthrough,
		"#partial":     TokenPartial,
		"#force":       TokenForce,
	}

	AssignBinOps = []TokenKind{
		TokenAssignSlash,
		TokenAssignMultiply,
		TokenAssignModule,
		TokenAssignMinus,
		TokenAssignPlus,
		TokenAssignOr,
		TokenAssignBitOr,
		TokenAssignAnd,
		TokenAssignBitAnd,
		TokenAssignBitLeftShift,
		TokenAssignBitRightShift,
		TokenAssignBitXor,
	}

	AssignOp = []TokenKind{
		TokenAssign,
		TokenWalrus,
		TokenBind,
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
		TokenBitAnd:      "&",
		TokenMultiply:    "*",
	}
)
