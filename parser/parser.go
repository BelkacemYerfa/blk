package parser

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
)

type (
	prefixParseFn func() Expression
	infixParseFn  func(Expression) Expression
)

const (
	_ int = iota
	LOWEST
	EQUALS      // == !=
	LESSGREATER // > < >= <=
	SUM         // +
	PRODUCT     // *
	PREFIX      // -X or !X
	CALL        // myFunction(X)
)

var precedences = map[TokenKind]int{
	TokenEquals:         EQUALS,
	TokenNotEquals:      EQUALS,
	TokenLess:           LESSGREATER,
	TokenLessOrEqual:    LESSGREATER,
	TokenGreater:        LESSGREATER,
	TokenGreaterOrEqual: LESSGREATER,
	TokenPlus:           SUM,
	TokenMinus:          SUM,
	TokenSlash:          PRODUCT,
	TokenMultiply:       PRODUCT,
}

func NewParser(tokens []Token, filepath string) *Parser {
	p := Parser{
		Tokens:         tokens,
		FilePath:       filepath,
		Errors:         []error{},
		prefixParseFns: make(map[TokenKind]prefixParseFn),
		infixParseFns:  make(map[TokenKind]infixParseFn),
		Pos:            0,
	}

	// prefix/unary operators
	p.registerPrefix(TokenIdentifier, p.parseIdentifier)
	p.registerPrefix(TokenInt, p.parseIntLiteral)
	p.registerPrefix(TokenFloat, p.parseFloatLiteral)
	p.registerPrefix(TokenExclamation, p.parsePrefixExpression)
	p.registerPrefix(TokenMinus, p.parsePrefixExpression)

	// infix/binary operators
	p.registerInfix(TokenPlus, p.parseInfixExpression)
	p.registerInfix(TokenMinus, p.parseInfixExpression)
	p.registerInfix(TokenSlash, p.parseInfixExpression)
	p.registerInfix(TokenMultiply, p.parseInfixExpression)
	p.registerInfix(TokenEquals, p.parseInfixExpression)
	p.registerInfix(TokenNotEquals, p.parseInfixExpression)
	p.registerInfix(TokenLess, p.parseInfixExpression)
	p.registerInfix(TokenGreater, p.parseInfixExpression)
	p.registerInfix(TokenLessOrEqual, p.parseInfixExpression)
	p.registerInfix(TokenGreaterOrEqual, p.parseInfixExpression)
	return &p
}

func (p *Parser) peekPrecedence() int {
	if p, ok := precedences[p.peekToken().Kind]; ok {
		return p
	}
	return LOWEST
}

func (p *Parser) nextToken() Token {
	if p.Pos >= len(p.Tokens) {
		return Token{LiteralToken: LiteralToken{Kind: TokenEOF}}
	}
	tok := p.Tokens[p.Pos]
	p.Pos++
	return tok
}

// Returns the current token to process, if none, returns the EOF
func (p *Parser) peekToken() Token {
	if p.Pos >= len(p.Tokens) {
		return Token{LiteralToken: LiteralToken{Kind: TokenEOF}}
	}
	return p.Tokens[p.Pos]
}

func (p *Parser) expect(kinds []TokenKind) (Token, error) {
	tok := p.nextToken()

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
	// skip until the next useful line
	for p.peekToken().Row <= tok.Row {
		p.nextToken()
	}
	return errors.New(errMsg)
}

func (p *Parser) registerPrefix(tokenType TokenKind, fn prefixParseFn) {
	p.prefixParseFns[tokenType] = fn
}
func (p *Parser) registerInfix(tokenType TokenKind, fn infixParseFn) {
	p.infixParseFns[tokenType] = fn
}

func (p *Parser) Parse() *Program {
	ast := Program{}
	ast.Statements = make([]Statement, 0)

	for p.peekToken().Kind != TokenEOF {

		stmt, err := p.parseStatement()

		if err != nil {
			p.Errors = append(p.Errors, err)
		} else {
			ast.Statements = append(ast.Statements, stmt)
		}

	}

	return &ast
}

func (p *Parser) parseStatement() (Statement, error) {
	cmdToken := p.peekToken() // Consume command
	switch cmdToken.Kind {
	case TokenLet, TokenVar:
		return p.parseLetStatement()
	case TokenReturn:
		return p.parseReturnStatement()
	default:
		return p.parseExpressionStatement()
	}
}

func (p *Parser) parseLetStatement() (*LetStatement, error) {
	stmt := &LetStatement{Token: p.peekToken()}
	p.nextToken()
	identifier := p.parseIdentifier()
	if identifier == nil {
		return nil, p.Errors[len(p.Errors)-1]
	}

	stmt.Name = identifier.(*Identifier)
	tok := p.nextToken()

	if tok.Kind != TokenColon {
		return nil, p.error(tok, "ERROR: expected colon (:), got shit")
	}

	tok = p.nextToken()

	if tok.Kind != TokenIdentifier {
		return nil, p.error(tok, "ERROR: expected type, got shit")
	}

	for p.peekToken().Row <= tok.Row {
		p.nextToken()
	}

	return stmt, nil
}

func (p *Parser) parseReturnStatement() (*ReturnStatement, error) {
	stmt := &ReturnStatement{Token: p.peekToken()}
	tok := p.nextToken()

	for p.peekToken().Row <= tok.Row {
		p.nextToken()
	}
	return stmt, nil
}

func (p *Parser) parseExpressionStatement() (*ExpressionStatement, error) {
	stmt := &ExpressionStatement{}
	stmt.Expression = p.parseExpression(LOWEST)
	return stmt, nil
}

func (p *Parser) parseIdentifier() Expression {
	tok := p.nextToken()

	if tok.Kind != TokenIdentifier {
		p.Errors = append(p.Errors, p.error(tok, "ERROR: expected identifier, got shit"))
		return nil
	}

	return &Identifier{
		Token: tok,
		Value: tok.Text,
	}
}

func (p *Parser) parseIntLiteral() Expression {
	tok := p.nextToken()

	num, err := strconv.ParseInt(tok.Text, 0, 64)
	if err != nil {
		return nil
	}
	return &IntegerLiteral{
		Token: tok,
		Value: num,
	}
}

func (p *Parser) parseFloatLiteral() Expression {
	tok := p.nextToken()
	num, err := strconv.ParseFloat(tok.Text, 64)
	if err != nil {
		return nil
	}
	return &FloatLiteral{
		Token: tok,
		Value: num,
	}
}

func (p *Parser) parsePrefixExpression() Expression {
	tok := p.nextToken()

	if _, ok := unaryOperators[tok.Kind]; !ok {
		p.Errors = append(p.Errors, p.error(tok, "ERROR: expected a unary operator (! | -), got shut"))
		return nil
	}

	right := p.parseExpression(PREFIX)

	return &UnaryExpression{
		Token:    tok,
		Operator: tok.Text,
		Right:    right,
	}
}

func (p *Parser) parseInfixExpression(left Expression) Expression {
	tok := p.peekToken()

	if _, ok := binOperators[tok.Kind]; !ok {
		p.Errors = append(p.Errors, p.error(tok, "ERROR: expected a binary operator (== | > | < | ...), got shut"))
		return nil
	}

	precedence := p.peekPrecedence()
	p.nextToken()
	right := p.parseExpression(precedence)

	return &BinaryExpression{
		Token:    tok,
		Operator: tok.Text,
		Left:     left,
		Right:    right,
	}
}

func (p *Parser) parseExpression(precedence int) Expression {
	prefix := p.prefixParseFns[p.peekToken().Kind]
	if prefix == nil {
		p.Errors = append(p.Errors, fmt.Errorf("ERROR: %v kind ain't supported", p.peekToken().Kind))
		return nil
	}

	leftExp := prefix()
	cur := p.peekToken()
	for p.peekToken().Row <= cur.Row && p.peekToken().Kind != TokenEOF && precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken().Kind]
		if infix == nil {
			return leftExp
		}
		leftExp = infix(leftExp)
	}

	return leftExp
}
