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
	INDEX       // arr[i]
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
	TokenBraceOpen:      CALL,
	TokenBracketOpen:    INDEX,
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
	p.registerPrefix(TokenString, p.parseStringLiteral)
	p.registerPrefix(TokenBracketOpen, p.parseArrayLiteral)
	p.registerPrefix(TokenExclamation, p.parsePrefixExpression)
	p.registerPrefix(TokenMinus, p.parsePrefixExpression)
	p.registerPrefix(TokenTrue, p.parseBooleanLiteral)
	p.registerPrefix(TokenFalse, p.parseBooleanLiteral)
	p.registerPrefix(TokenBraceOpen, p.parseGroupedExpression)
	p.registerPrefix(TokenIf, p.parseIfExpression)

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
	p.registerInfix(TokenBraceOpen, p.parseCallExpression)
	p.registerInfix(TokenBracketOpen, p.parseIndexExpression)
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

func (p *Parser) expect(kinds []TokenKind) bool {
	tok := p.nextToken()
	if slices.Index(kinds, tok.Kind) == -1 {
		p.Errors = append(p.Errors, p.error(tok, fmt.Sprintf("ERROR: expected one of (%v), received %v", kinds, tok.Kind)))
		return false
	}

	return true
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
		if key, isMatching := keywords[tok.Text]; isMatching && key != TokenFalse && key != TokenTrue {
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
		// tok := p.peekToken()

		// switch tok.Kind {
		// case TokenLet, TokenVar, TokenReturn:
		stmt, err := p.parseStatement()

		if err != nil {
			p.Errors = append(p.Errors, err)
			return nil
		} else {
			ast.Statements = append(ast.Statements, stmt)
		}
		// default:
		// 	p.Errors = append(p.Errors, p.error(tok, "ERROR: expected a stmt, got shit"))
		// 	return nil
		// }
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
	case TokenFn:
		return p.parseFunctionStatement()
	case TokenIdentifier:
		return p.parseAssignmentStatement()
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

	stmt.ExplicitType = p.parseType()

	tok = p.peekToken()
	fmt.Println(tok)
	if tok.Kind != TokenAssign {
		return nil, p.error(tok, "ERROR: expected assign (=), got shit")
	}

	p.nextToken()
	stmt.Value = p.parseExpression(LOWEST)

	return stmt, nil
}

func (p *Parser) parseReturnStatement() (*ReturnStatement, error) {
	stmt := &ReturnStatement{Token: p.peekToken()}
	p.nextToken()
	stmt.ReturnValue = p.parseExpression(LOWEST)
	return stmt, nil
}

func (p *Parser) parseExpressionStatement() (*ExpressionStatement, error) {
	stmt := &ExpressionStatement{Token: p.peekToken()}
	expr := p.parseExpression(LOWEST)
	if expr == nil {
		return nil, fmt.Errorf("ERROR: on the expression stmt")
	}
	stmt.Expression = expr
	return stmt, nil
}

func (p *Parser) parseAssignmentStatement() (*AssignmentStatement, error) {
	stmt := &AssignmentStatement{Token: p.peekToken()}

	stmt.Name = p.parseExpression(LOWEST)

	// if !p.expect([]TokenKind{TokenIdentifier}) {
	// 	p.Pos--
	// 	errMsg := fmt.Sprintf("ERROR: expected identifier, got shit")
	// 	tok := p.peekToken()
	// 	p.Pos++
	// 	return nil, p.error(tok, errMsg)
	// }

	tok := p.nextToken()

	if tok.Kind != TokenAssign {
		return nil, p.error(tok, "ERROR: expected assignment (=), got shit")
	}

	value := p.parseExpression(LOWEST)

	if value == nil {
		return nil, fmt.Errorf("ERROR: on the expression stmt")
	}

	stmt.Value = value
	return stmt, nil
}

func (p *Parser) parseType() TYPE {
	tok := p.nextToken()
	switch tok.Kind {
	case TokenIdentifier:
		p.Pos--
		tok = p.peekToken()
		p.Pos++
		return p.typeMapper(tok.Text)
	case TokenBracketOpen:
		tok = p.peekToken()

		for tok.Kind != TokenIdentifier {
			tok = p.nextToken()
		}

		returnedType := p.typeMapper(tok.Text)

		p.nextToken()
		for tok.Kind == TokenBracketClose {
			tok = p.nextToken()
		}
		p.nextToken()
		return returnedType
	default:
		errMsg := p.error(tok, "ERROR: expected type, got shit")
		panic(errMsg)
	}
}

func (p *Parser) typeMapper(typ string) TYPE {
	switch typ {
	case "int":
		return IntType
	case "float":
		return FloatType
	case "string":
		return StringType
	case "bool":
		return BoolType
	case "void":
		return VoidType
	}

	panic(fmt.Sprintf("ERROR: type ain't supported (%v)", typ))
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

func (p *Parser) parseStringLiteral() Expression {
	tok := p.nextToken()
	return &StringLiteral{
		Token: tok,
		Value: tok.Text,
	}
}

func (p *Parser) parseBooleanLiteral() Expression {
	tok := p.nextToken()
	truth := tok.Text == "true"
	return &BooleanLiteral{
		Token: tok,
		Value: truth,
	}
}

func (p *Parser) parseArrayLiteral() Expression {
	prev := p.peekToken()
	if !p.expect([]TokenKind{TokenBracketOpen}) {
		p.Errors = append(p.Errors, p.error(prev, "ERROR: expected open bracket [, got shit"))
		return nil
	}
	elements := make([]Expression, 0)

	tok := p.peekToken()

	if tok.Kind == TokenBracketClose {
		p.nextToken()
		return &ArrayLiteral{
			Token:    prev,
			Elements: elements,
		}
	}

	elements = append(elements, p.parseExpression(LOWEST))

	for p.peekToken().Kind == TokenComma {
		p.nextToken()
		elements = append(elements, p.parseExpression(LOWEST))
	}

	if !p.expect([]TokenKind{TokenBracketClose}) {
		p.Errors = append(p.Errors, p.error(prev, "ERROR: expected close bracket ( ] ), got shit"))
		return nil
	}

	return &ArrayLiteral{
		Token:    prev,
		Elements: elements,
	}
}

func (p *Parser) parseGroupedExpression() Expression {
	p.nextToken()
	exp := p.parseExpression(LOWEST)
	if !p.expect([]TokenKind{TokenBraceClose}) {
		return nil
	}
	return exp
}

func (p *Parser) parseIfExpression() Expression {
	prev := p.peekToken()
	p.nextToken()

	condition := p.parseExpression(LOWEST)

	if !p.expect([]TokenKind{TokenCurlyBraceOpen}) {
		return nil
	}

	consequence := p.parseBlockStatement().(*BlockStatement)

	// check if there is an else stmt
	tok := p.nextToken()
	alternative := &BlockStatement{}
	if tok.Kind == TokenElse {
		if !p.expect([]TokenKind{TokenCurlyBraceOpen}) {
			return nil
		}
		alternative = p.parseBlockStatement().(*BlockStatement)
	} else {
		p.Pos--
	}

	return &IfExpression{
		Token:       prev,
		Condition:   condition,
		Consequence: consequence,
		Alternative: alternative,
	}
}

func (p *Parser) parseFunctionStatement() (*FunctionStatement, error) {
	prev := p.peekToken()
	p.nextToken()

	if !p.expect([]TokenKind{TokenIdentifier}) {
		return nil, fmt.Errorf("ERROR: expected identifier, got shit")
	}
	p.Pos--
	name := p.peekToken().Text
	p.nextToken()
	if !p.expect([]TokenKind{TokenBraceOpen}) {
		return nil, fmt.Errorf("ERROR: expected brace open ( ( ), got shit")
	}

	args := p.parseArguments()

	if !p.expect([]TokenKind{TokenColon}) {
		return nil, fmt.Errorf("ERROR: expected colon ( : ), got shit")
	}

	returnType := p.parseType()

	if !p.expect([]TokenKind{TokenCurlyBraceOpen}) {
		return nil, fmt.Errorf("ERROR: expected curly brace open ( { ), got shit")
	}

	body := p.parseBlockStatement().(*BlockStatement)

	return &FunctionStatement{
		Token:      prev,
		Name:       name,
		Args:       args,
		ReturnType: returnType,
		Body:       body,
	}, nil
}

func (p *Parser) parseArguments() []*Identifier {
	args := make([]*Identifier, 0)

	if p.peekToken().Kind == TokenBraceClose {
		p.nextToken()
		return args
	}

	args = append(args, &Identifier{
		Token: p.peekToken(),
		Value: p.peekToken().Text,
	})

	p.nextToken()
	if !p.expect([]TokenKind{TokenColon}) {
		return nil
	}

	if !p.expect([]TokenKind{TokenIdentifier}) {
		return nil
	}

	for p.peekToken().Kind == TokenComma {
		p.nextToken()
		args = append(args, &Identifier{
			Token: p.peekToken(),
			Value: p.peekToken().Text,
		})

		p.nextToken()
		if !p.expect([]TokenKind{TokenColon}) {
			return nil
		}

		if !p.expect([]TokenKind{TokenIdentifier}) {
			return nil
		}
	}

	if !p.expect([]TokenKind{TokenBraceClose}) {
		return nil
	}

	return args
}

func (p *Parser) parseBlockStatement() Expression {
	block := BlockStatement{Token: p.peekToken()}
	block.Body = make([]Statement, 0)
	for p.peekToken().Kind != TokenCurlyBraceClose && p.peekToken().Kind != TokenEOF {
		// parse body expressions and statements
		stmt, err := p.parseStatement()
		if err != nil {
			p.Errors = append(p.Errors, err)
		} else {
			block.Body = append(block.Body, stmt)
		}
	}

	if !p.expect([]TokenKind{TokenCurlyBraceClose}) {
		return nil
	}

	return &block
}

func (p *Parser) parseCallExpression(left Expression) Expression {
	switch left.(type) {
	case *Identifier:
	default:
		p.Errors = append(p.Errors, p.error(p.peekToken(), "ERROR: only call are allowed, bounding function into a variable ain't allowed"))
		return nil
	}

	exp := CallExpression{Token: p.peekToken(), Function: *(left.(*Identifier))}
	exp.Args = p.parseCallArguments()
	return &exp
}

func (p *Parser) parseCallArguments() []Expression {
	args := make([]Expression, 0)
	if !p.expect([]TokenKind{TokenBraceOpen}) {
		return nil
	}

	if p.peekToken().Kind == TokenBraceClose {
		p.nextToken()
		return args
	}

	args = append(args, p.parseExpression(LOWEST))

	for p.peekToken().Kind == TokenComma {
		p.nextToken()
		args = append(args, p.parseExpression(LOWEST))
	}

	if !p.expect([]TokenKind{TokenBraceClose}) {
		return nil
	}

	return args
}

func (p *Parser) parseIndexExpression(left Expression) Expression {
	exp := &IndexExpression{Token: p.peekToken(), Left: left}

	p.nextToken()

	exp.Index = p.parseExpression(LOWEST)

	if !p.expect([]TokenKind{TokenBracketClose}) {
		return nil
	}

	return exp
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
	for p.peekToken().Row <= cur.Row && precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken().Kind]
		if infix == nil {
			return leftExp
		}
		leftExp = infix(leftExp)
	}

	return leftExp
}
