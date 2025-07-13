package parser

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strconv"
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
			if t.Kind == TokenString {
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

func (p *Parser) Parse() *Program {
	ast := Program{}
	ast.Statements = make([]Statement, 0)

	for p.peek().Kind != TokenEOF {
		tok := p.peek()
		switch tok.Kind {
		case TokenLet, TokenVar:
			stmt, err := p.parseStatement()

			if err != nil {
				fmt.Println(err)
				return nil
			} else {
				ast.Statements = append(ast.Statements, stmt)
			}
		}
	}

	return &ast
}

func (p *Parser) parseStatement() (Statement, error) {
	cmdToken := p.next() // Consume command
	switch cmdToken.Kind {
	case TokenLet:
		return p.parseLetStatement()
	}
	// All good, create AST node
	return nil, fmt.Errorf("ERROR: unexpected token appeared, line %v row%v", cmdToken.Row, cmdToken.Col)
}

func (p *Parser) parseLetStatement() (*LetStatement, error) {

	identifier, err := p.ParseIdentifier()
	if err != nil {
		return nil, err
	}

	tok := p.next()

	if tok.Kind != TokenColon {
		return nil, p.error(tok, "ERROR: expected colon (:), got shit")
	}

	tok = p.next()

	if tok.Kind != TokenIdentifier {
		return nil, p.error(tok, "ERROR: expected a type, got shit")
	}

	tok = p.next()

	if tok.Kind != TokenEqual {
		return nil, p.error(tok, "ERROR: expected an assignment (=), got shit")
	}

	value := p.parseExpression()

	return &LetStatement{
		Name:  identifier,
		Value: value,
	}, nil
}

func (p *Parser) ParseIdentifier() (*Identifier, error) {
	tok := p.next()

	if tok.Kind != TokenIdentifier {
		return nil, p.error(tok, "ERROR: expected identifier, got shit")
	}

	identifier := Identifier{}
	identifier.Token = tok
	identifier.Value = tok.Text
	return &identifier, nil
}

func (p *Parser) parseExpression() Expression {
	key := p.next()

	switch key.Kind {
	case TokenNumber:
		tok := p.next()
		switch tok.Kind {
		case TokenPlus, TokenSlash, TokenMultiply, TokenMinus:
			return p.parseBinaryExpression(key)
		default:
			p.Pos--
			return p.parseNumberLiteral(key)
		}
	case TokenString:
		tok := p.next()
		switch tok.Kind {
		case TokenPlus, TokenSlash, TokenMultiply, TokenMinus:
			return p.parseBinaryExpression(key)
		default:
			p.Pos--
			return p.parseStringLiteral(key)
		}
	default:
		p.Pos--
	}
	return nil
}

func (p *Parser) parseBinaryExpression(prev Token) *BinaryExpression {
	operator := p.peek().Text

	left := p.parseNumberLiteral(prev)
	right := p.parseExpression()

	return &BinaryExpression{
		Token:    p.peek(),
		Operator: operator,
		Left:     left,
		Right:    right,
	}
}

func (p *Parser) parseNumberLiteral(prev Token) *LiteralExpression {
	num, err := strconv.ParseFloat(prev.Text, 64)
	if err != nil {
		return nil
	}
	return &LiteralExpression{
		Token: prev,
		Value: num,
	}
}

func (p *Parser) parseStringLiteral(prev Token) *LiteralExpression {
	return &LiteralExpression{
		Token: prev,
		Value: prev.Text,
	}
}
