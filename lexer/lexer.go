package lexer

import (
	"fmt"
	"os"
	"strings"
	"unicode"
)

func NewLexer(filePath string, content string) *Lexer {
	lexer := Lexer{
		Content:  []rune(content),
		FilePath: filePath,
		Row:      1,
		Col:      1,
		Cur:      0,
	}
	return &lexer
}

func (l *Lexer) readChar() {
	if l.Cur >= len(l.Content) {
		// reach end of file
		l.Cur = 0
		return
	}

	char := l.Content[l.Cur]

	switch char {
	case '\n':
		l.Row++
		l.Col = 1
	default:
		l.Col++
	}

	// increment to deal with the next char
	l.Cur++
}

type Token struct {
	LiteralToken
	Row int
	Col int
}

func (l *Lexer) NextToken() Token {
	l.skipWhiteSpace()
	l.skipComment()

	token := Token{
		Row: l.Row,
		Col: l.Col,
	}

	if l.Cur >= len(l.Content) {
		token.LiteralToken = LiteralToken{
			Kind: TokenEOF,
			Text: "",
		}
		return token
	}

	char := l.Content[l.Cur]

	switch string(char) {
	case TokenCurlyBraceOpen:
		l.readChar()
		token.LiteralToken = LiteralToken{
			Kind: TokenCurlyBraceOpen,
			Text: "{",
		}
	case TokenCurlyBraceClose:
		l.readChar()
		token.LiteralToken = LiteralToken{
			Kind: TokenCurlyBraceClose,
			Text: "}",
		}
	case TokenBraceOpen:
		l.readChar()
		token.LiteralToken = LiteralToken{
			Kind: TokenBraceOpen,
			Text: "(",
		}
	case TokenBraceClose:
		l.readChar()
		token.LiteralToken = LiteralToken{
			Kind: TokenBraceClose,
			Text: ")",
		}
	case TokenBracketOpen:
		l.readChar()
		token.LiteralToken = LiteralToken{
			Kind: TokenBracketOpen,
			Text: "[",
		}
	case TokenBracketClose:
		l.readChar()
		token.LiteralToken = LiteralToken{
			Kind: TokenBracketClose,
			Text: "]",
		}
	case TokenColon:
		l.readChar()
		nextChar := string(l.Content[l.Cur])
		switch nextChar {
		case ":":
			l.readChar()
			token.LiteralToken = LiteralToken{
				Kind: TokenBind,
				Text: "::",
			}
		case "=":
			l.readChar()
			token.LiteralToken = LiteralToken{
				Kind: TokenWalrus,
				Text: ":=",
			}
		default:
			token.LiteralToken = LiteralToken{
				Kind: TokenColon,
				Text: ":",
			}
		}
	case TokenDot:
		l.readChar()
		token.LiteralToken = LiteralToken{
			Kind: TokenDot,
			Text: ".",
		}
	case TokenComma:
		l.readChar()
		token.LiteralToken = LiteralToken{
			Kind: TokenComma,
			Text: ",",
		}
	case TokenMinus:
		l.readChar()
		nextChar := string(l.Content[l.Cur])
		switch nextChar {
		case "=":
			l.readChar()
			token.LiteralToken = LiteralToken{
				Kind: TokenAssignMinus,
				Text: "-=",
			}
		case "-":
			l.readChar()
			token.LiteralToken = LiteralToken{
				Kind: TokenAssignMinusOne,
				Text: "--",
			}
		default:
			token.LiteralToken = LiteralToken{
				Kind: TokenMinus,
				Text: "-",
			}
		}
	case TokenPlus:
		l.readChar()
		nextChar := string(l.Content[l.Cur])
		switch nextChar {
		case "=":
			l.readChar()
			token.LiteralToken = LiteralToken{
				Kind: TokenAssignPlus,
				Text: "+=",
			}
		case "+":
			l.readChar()
			token.LiteralToken = LiteralToken{
				Kind: TokenAssignPlusOne,
				Text: "++",
			}
		default:
			token.LiteralToken = LiteralToken{
				Kind: TokenPlus,
				Text: "+",
			}
		}
	case TokenMultiply:
		l.readChar()
		equalsChar := string(l.Content[l.Cur])
		if equalsChar == TokenAssign {
			l.readChar()
			token.LiteralToken = LiteralToken{
				Kind: TokenAssignMultiply,
				Text: "*=",
			}
		} else {
			token.LiteralToken = LiteralToken{
				Kind: TokenMultiply,
				Text: "*",
			}
		}
	case TokenModule:
		l.readChar()
		equalsChar := string(l.Content[l.Cur])
		if equalsChar == TokenAssign {
			l.readChar()
			token.LiteralToken = LiteralToken{
				Kind: TokenAssignModule,
				Text: "%=",
			}
		} else {
			token.LiteralToken = LiteralToken{
				Kind: TokenModule,
				Text: "%",
			}
		}
	case TokenSlash:
		l.readChar()
		equalsChar := string(l.Content[l.Cur])
		if equalsChar == TokenAssign {
			l.readChar()
			token.LiteralToken = LiteralToken{
				Kind: TokenAssignSlash,
				Text: "/=",
			}
		} else {
			token.LiteralToken = LiteralToken{
				Kind: TokenSlash,
				Text: "/",
			}
		}
	case TokenExclamation:
		l.readChar()
		equalChar := string(l.Content[l.Cur])
		if equalChar == TokenAssign {
			l.readChar()
			token.LiteralToken = LiteralToken{
				Kind: TokenNotEquals,
				Text: "!=",
			}
		} else {
			token.LiteralToken = LiteralToken{
				Kind: TokenExclamation,
				Text: "!",
			}
		}
	case TokenAssign:
		l.readChar()
		nextChar := string(l.Content[l.Cur])
		switch nextChar {
		case TokenAssign:
			l.readChar()
			token.LiteralToken = LiteralToken{
				Kind: TokenEquals,
				Text: "==",
			}
		case TokenGreater:
			l.readChar()
			token.LiteralToken = LiteralToken{
				Kind: TokenMatch,
				Text: "=>",
			}
		default:
			token.LiteralToken = LiteralToken{
				Kind: TokenAssign,
				Text: "=",
			}
		}
	case TokenGreater:
		l.readChar()
		nextChar := string(l.Content[l.Cur])
		if nextChar == TokenAssign {
			l.readChar()
			token.LiteralToken = LiteralToken{
				Kind: TokenGreaterOrEqual,
				Text: ">=",
			}
		} else if isLetter(char) {
			return l.readIdentifier()
		} else if isDigit(char) {
			return l.readNumber()
		} else {
			token.LiteralToken = LiteralToken{
				Kind: TokenGreater,
				Text: ">",
			}
		}
	case TokenLess:
		l.readChar()
		nextChar := string(l.Content[l.Cur])
		if nextChar == TokenAssign {
			l.readChar()
			token.LiteralToken = LiteralToken{
				Kind: TokenGreaterOrEqual,
				Text: "<=",
			}
		} else if isLetter(char) {
			return l.readIdentifier()
		} else if isDigit(char) {
			return l.readNumber()
		} else {
			token.LiteralToken = LiteralToken{
				Kind: TokenLess,
				Text: "<",
			}
		}
	case "&":
		l.readChar()
		nextChar := string(l.Content[l.Cur])
		if nextChar == "&" {
			l.readChar()
			nextChar := string(l.Content[l.Cur])
			if nextChar == "=" {
				l.readChar()
				token.LiteralToken = LiteralToken{
					Kind: TokenAssignAnd,
					Text: "&&=",
				}
			} else {
				token.LiteralToken = LiteralToken{
					Kind: TokenAnd,
					Text: "&&",
				}
			}
		} else {
			token.LiteralToken = LiteralToken{
				Kind: TokenError,
				Text: nextChar,
			}
		}
	case "|":
		l.readChar()
		nextChar := string(l.Content[l.Cur])
		if nextChar == "|" {
			l.readChar()
			nextChar := string(l.Content[l.Cur])
			if nextChar == "=" {
				l.readChar()
				token.LiteralToken = LiteralToken{
					Kind: TokenAssignOr,
					Text: "||=",
				}
			} else {
				token.LiteralToken = LiteralToken{
					Kind: TokenOr,
					Text: "||",
				}
			}
		} else {
			token.LiteralToken = LiteralToken{
				Kind: TokenError,
				Text: nextChar,
			}
		}
	case TokenQuote:
		return l.readString()
	case TokenEOF:
		l.readChar()
		token.LiteralToken = LiteralToken{
			Kind: TokenEOF,
			Text: "",
		}
	default:
		if isLetter(char) {
			return l.readIdentifier()
		} else if isDigit(char) {
			return l.readNumber()
		} else {
			l.readChar()
			token.LiteralToken = LiteralToken{
				Kind: TokenError,
				Text: string(char),
			}
		}
	}
	return token
}

func (l *Lexer) Tokenize() []Token {
	var tokens []Token
	for {
		tok := l.NextToken()
		tokens = append(tokens, tok)
		if tok.Kind == TokenEOF {
			break
		}
	}
	return tokens
}

func isLetter(char rune) bool {
	// Accept common path and identifier characters
	return unicode.IsLetter(char) || char == '_'
}

func isDigit(char rune) bool {
	return unicode.IsDigit(char)
}

func (l *Lexer) readIdentifier() Token {
	startPos := l.Cur

	// save them to return
	row := l.Row
	col := l.Col

	for l.Cur < len(l.Content) {
		char := l.Content[l.Cur]
		if isLetter(char) || isDigit(char) {
			l.readChar()
		} else {
			break
		}
	}

	text := strings.TrimSpace(string(l.Content[startPos:l.Cur]))

	if tokenKind, isKeyword := Keywords[text]; isKeyword {
		return Token{LiteralToken: LiteralToken{
			Kind: tokenKind,
			Text: text,
		}, Row: row, Col: col}
	}

	return Token{
		LiteralToken: LiteralToken{
			Kind: TokenIdentifier,
			Text: string(text),
		},
		Row: row,
		Col: col,
	}
}

func (l *Lexer) readString() Token {
	start := l.Cur + 1 // skip the opening quote
	row, col := l.Row, l.Col

	l.Cur++

	for l.Cur < len(l.Content) && l.Content[l.Cur] != '"' {
		l.readChar()
	}

	if l.Cur >= len(l.Content) {
		fmt.Println(`ERROR: the quoted data, doesn't have a closing Quote (")`)
		os.Exit(1)
	}

	end := l.Cur
	l.readChar() // consume the closing quote

	text := strings.TrimSpace(string(l.Content[start:end]))

	return Token{
		LiteralToken: LiteralToken{
			Kind: TokenString,
			Text: text,
		},
		Row: row,
		Col: col,
	}
}

func (l *Lexer) readNumber() Token {
	startPos := l.Cur
	row := l.Row
	col := l.Col

	// Read integer part
	for l.Cur < len(l.Content) && isDigit(l.Content[l.Cur]) {
		l.readChar()
	}

	if l.Cur < len(l.Content) && l.Content[l.Cur] == '.' {
		if l.Cur < len(l.Content) && l.Content[l.Cur] == '.' {
			l.readChar() // consume '.'

			// Read fractional part
			for l.Cur < len(l.Content) && isDigit(l.Content[l.Cur]) {
				l.readChar()
			}
		}
		text := string(l.Content[startPos:l.Cur])
		return Token{
			LiteralToken: LiteralToken{
				Kind: TokenFloat,
				Text: text,
			},
			Row: row,
			Col: col,
		}
	} else {
		text := string(l.Content[startPos:l.Cur])

		return Token{
			LiteralToken: LiteralToken{
				Kind: TokenInt,
				Text: text,
			},
			Row: row,
			Col: col,
		}
	}

}

func (l *Lexer) skipComment() {
	for l.Cur < len(l.Content) && l.Content[l.Cur] == '#' {
		for l.Cur < len(l.Content) && l.Content[l.Cur] != '\n' {
			l.readChar()
		}
		l.skipWhiteSpace()
	}
}

func (l *Lexer) skipWhiteSpace() {
	for l.Cur < len(l.Content) && unicode.IsSpace(l.Content[l.Cur]) {
		l.readChar()
	}
}
