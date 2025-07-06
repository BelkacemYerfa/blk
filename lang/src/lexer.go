package src

import (
	"strings"
	"time"
	"unicode"
)

type TokenKind = string

const (

	// Commands
	TokenPush          TokenKind = "push"
	TokenTrim          TokenKind = "trim"
	TokenExport        TokenKind = "export"
	TokenThumbnailFrom TokenKind = "thumbnail_from"
	TokenConcat        TokenKind = "concat"
	TokenSetTrack      TokenKind = "set_track"
	TokenUseTrack      TokenKind = "use_track"

	// Block Units
	TokenCurlyBraceOpen  TokenKind = "{"
	TokenCurlyBraceClose TokenKind = "}"
	TokenQuote           TokenKind = `"`
	TokenColon           TokenKind = ":"

	// Comment
	TokenComment TokenKind = "#"

	// Var Naming
	TokenIdentifier TokenKind = "identifier"

	// Var Types
	TokenString TokenKind = "string"
	TokenTime   TokenKind = "time"
	TokenNumber TokenKind = "number"
	TokenBool   TokenKind = "bool"

	// Error
	TokenError TokenKind = "error"

	// EOF
	TokenEOF TokenKind = "end of file"
)

type LiteralToken struct {
	Text string
	Kind TokenKind
}

type Lexer struct {
	Content []rune
	// help mainly in error detection when having multi file execution
	FilePath string
	Row      int
	Col      int
	Cur      int
}

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
	case TokenColon:
		l.readChar()
		return Token{
			LiteralToken: LiteralToken{
				Kind: TokenColon,
				Text: ":",
			},
			Row: l.Row,
			Col: l.Col,
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
				Text: "unexpected token encountered",
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

func checkTimeTrimFormatValid(tm string) bool {
	_, err := time.Parse("15:04:05", tm)
	if err != nil {
		return false
	}
	return true
}

var keywords = map[string]TokenKind{
	"set_track":      TokenSetTrack,
	"push":           TokenPush,
	"use_track":      TokenUseTrack,
	"export":         TokenExport,
	"trim":           TokenTrim,
	"thumbnail_from": TokenThumbnailFrom,
	"concat":         TokenConcat,
	"true":           TokenBool,
	"false":          TokenBool,
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

	if tokenKind, isKeyword := keywords[text]; isKeyword {
		return Token{LiteralToken: LiteralToken{
			Kind: tokenKind,
			Text: text,
		}, Row: row, Col: col}
	}

	// check for boolean time token
	if text == "true" || text == "false" {
		return Token{LiteralToken: LiteralToken{Kind: TokenBool, Text: text}, Row: row, Col: col}
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

	for {
		l.readChar()
		if l.Cur >= len(l.Content) || l.Content[l.Cur] == '"' {
			break
		}
	}
	end := l.Cur
	l.readChar() // consume the closing quote

	text := strings.TrimSpace(string(l.Content[start:end]))

	// check the format if it is time format, that we support return a time token
	if checkTimeTrimFormatValid(text) {
		return Token{
			LiteralToken: LiteralToken{
				Kind: TokenTime,
				Text: text,
			},
			Row: row,
			Col: col,
		}
	}

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

	// Handle decimal point
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
			Kind: TokenNumber,
			Text: text,
		},
		Row: row,
		Col: col,
	}
}

func (l *Lexer) skipComment() {
	for l.Cur < len(l.Content) && l.Content[l.Cur] == '#' {
		for l.Cur < len(l.Content) && l.Content[l.Cur] != '\n' {
			l.readChar()
		}
		if l.Cur < len(l.Content) && l.Content[l.Cur] == '\n' {
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
