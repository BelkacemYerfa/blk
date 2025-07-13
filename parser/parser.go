package parser

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
)

func NewParser(tokens []Token, filepath string) *Parser {
	return &Parser{
		Tokens:   tokens,
		FilePath: filepath,
		Pos:      0,
	}
}

func (p *Parser) next() Token {
	if p.Pos >= len(p.Tokens) {
		return Token{LiteralToken: LiteralToken{Kind: TokenEOF}}
	}
	tok := p.Tokens[p.Pos]
	p.Pos++
	return tok
}

// Returns the current token to process, if none, returns the EOF
func (p *Parser) peek() Token {
	if p.Pos >= len(p.Tokens) {
		return Token{LiteralToken: LiteralToken{Kind: TokenEOF}}
	}
	return p.Tokens[p.Pos]
}

func (p *Parser) expect(kinds []TokenKind) (Token, error) {
	tok := p.next()

	if slices.Index(kinds, tok.Kind) == -1 {
		return tok, fmt.Errorf("ERROR: expected one of (%v), received %v", kinds, tok.Kind)
	}

	return tok, nil
}

func (p *Parser) error(tok Token, msg string) error {
	switch tok.Kind {
	case TokenCurlyBraceOpen, TokenCurlyBraceClose, TokenColon:
		tok = p.Tokens[p.Pos-2]
		tok.Col = tok.Col + len(tok.Text)
	case TokenEOF:
		tok = p.Tokens[p.Pos-2]
		tok.Col = tok.Col + len(tok.Text) + 1
	default:
		if key, isMatching := keywords[tok.Text]; isMatching && key != TokenBool {
			prev := p.Tokens[p.Pos-2]
			if tok.Row >= prev.Row {
				tok = prev
				tok.Col = tok.Col + len(tok.Text) + 1
			}
		}
	}

	errMsg := fmt.Sprintf("\033[1;90m%s:%d:%d:\033[0m\n\n", p.FilePath, tok.Row, tok.Col)

	// Build row set map
	rowSet := make(map[int][]Token)
	for _, t := range p.Tokens {
		rowSet[t.Row] = append(rowSet[t.Row], t)
	}

	// Collect sorted rows
	rows := []int{}
	for row := range rowSet {
		rows = append(rows, row)
	}
	sort.Ints(rows)

	// Find closest previous and next row
	var prevRow, nextRow int
	prevRow, nextRow = -1, -1
	for _, row := range rows {
		if row < tok.Row {
			prevRow = row
		} else if row > tok.Row && nextRow == -1 {
			nextRow = row
		}
	}

	// Build rowMap with only prevRow, tok.Row, nextRow
	rowMap := make(map[int][]Token)
	if prevRow != -1 {
		rowMap[prevRow] = rowSet[prevRow]
	}
	rowMap[tok.Row] = rowSet[tok.Row]
	if nextRow != -1 {
		rowMap[nextRow] = rowSet[nextRow]
	}

	// Format rows
	formattedRows := []int{}
	for row := range rowMap {
		formattedRows = append(formattedRows, row)
	}
	sort.Ints(formattedRows)

	for _, row := range formattedRows {
		currentLine := rowMap[row]
		lineContent := ""
		lastCol := 0

		for _, t := range currentLine {
			if t.Col > lastCol {
				lineContent += strings.Repeat(" ", t.Col-lastCol)
			}
			if t.Kind == TokenString || t.Kind == TokenTime {
				t.Text = fmt.Sprintf(`"%s"`, t.Text)
			}
			lineContent += t.Text
			lastCol = t.Col + len(t.Text)
		}

		lineNumStr := fmt.Sprintf("%d", row)
		errMsg += fmt.Sprintf("%s    %s\n", lineNumStr, lineContent)

		if row == tok.Row {
			spacesBeforeLineNum := len(lineNumStr)
			spacesAfterLineNum := 4
			spacesBeforeToken := tok.Col

			totalSpaces := spacesBeforeLineNum + spacesAfterLineNum + spacesBeforeToken

			errorIndicator := strings.Repeat(" ", totalSpaces)
			errMsg += errorIndicator + "\033[1;31m"
			repeat := len(tok.Text)
			if repeat == 0 {
				repeat = 1
			}
			errMsg += strings.Repeat("^", repeat)
			errMsg += "\033[0m\n"
		}
	}

	errMsg += msg
	return errors.New(errMsg)
}

func (p *Parser) Parse() *AST {
	ast := make(AST, 0)

	for p.Pos < len(p.Tokens) {
		tok := p.peek()
		fmt.Println(tok)
		p.next()
	}

	return &ast
}

func (p *Parser) parseCommand() (*StatementNode, error) {
	cmdToken := p.next() // Consume command

	// All good, create AST node
	return &StatementNode{}, fmt.Errorf("ERROR: unexpected token appeared, line %v row%v", cmdToken.Row, cmdToken.Col)
}
