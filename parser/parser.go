package parser

import (
	"blk/ast"
	"blk/lexer"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
)

const (
	_ int = iota
	LOWEST
	ASSIGN
	OR
	AND
	EQUALS      // == !=
	LESSGREATER // > < >= <=
	SUM         // +
	PRODUCT     // *
	PREFIX      // -X or !X
	CALL        // myFunction(X)
	INDEX       // arr[i]
	STRUCT      // myStruct { }
)

var precedences = map[lexer.TokenKind]int{
	lexer.TokenCurlyBraceOpen: ASSIGN,
	lexer.TokenAssign:         ASSIGN,
	lexer.TokenBind:           ASSIGN,
	lexer.TokenWalrus:         ASSIGN,
	lexer.TokenOr:             OR,
	lexer.TokenAnd:            AND,
	lexer.TokenEquals:         EQUALS,
	lexer.TokenNotEquals:      EQUALS,
	lexer.TokenLess:           LESSGREATER,
	lexer.TokenLessOrEqual:    LESSGREATER,
	lexer.TokenGreater:        LESSGREATER,
	lexer.TokenGreaterOrEqual: LESSGREATER,
	lexer.TokenPlus:           SUM,
	lexer.TokenMinus:          SUM,
	lexer.TokenSlash:          PRODUCT,
	lexer.TokenMultiply:       PRODUCT,
	lexer.TokenModule:         PRODUCT,
	lexer.TokenBraceOpen:      CALL,
	lexer.TokenBracketOpen:    INDEX,
	lexer.TokenDot:            STRUCT,
}

var AtomicTypes = map[string]ast.TYPE{
	"int":    ast.IntType,
	"float":  ast.FloatType,
	"string": ast.StringType,
	"bool":   ast.BoolType,
	"void":   ast.VoidType,
}

type (
	prefixParseFn func() ast.Expression
	infixParseFn  func(ast.Expression) ast.Expression
)

type Parser struct {
	Tokens         []lexer.Token
	FilePath       string
	Errors         []error
	Pos            int
	prefixParseFns map[lexer.TokenKind]prefixParseFn
	infixParseFns  map[lexer.TokenKind]infixParseFn
	internalFlags  []string
}

func NewParser(tokens []lexer.Token, filepath string) *Parser {
	p := Parser{
		Tokens:         tokens,
		FilePath:       filepath,
		Errors:         []error{},
		prefixParseFns: make(map[lexer.TokenKind]prefixParseFn),
		infixParseFns:  make(map[lexer.TokenKind]infixParseFn),
		Pos:            0,
		internalFlags:  []string{},
	}

	// prefix/unary operators
	p.registerPrefix(lexer.TokenIdentifier, p.parseIdentifier)
	p.registerPrefix(lexer.TokenInt, p.parseIntLiteral)
	p.registerPrefix(lexer.TokenFloat, p.parseFloatLiteral)
	p.registerPrefix(lexer.TokenString, p.parseStringLiteral)
	p.registerPrefix(lexer.TokenBracketOpen, p.parseArrayLiteral)
	p.registerPrefix(lexer.TokenCurlyBraceOpen, p.parseMapLiteral)
	p.registerPrefix(lexer.TokenExclamation, p.parsePrefixExpression)
	p.registerPrefix(lexer.TokenMinus, p.parsePrefixExpression)
	p.registerPrefix(lexer.TokenBool, p.parseBooleanLiteral)
	p.registerPrefix(lexer.TokenBraceOpen, p.parseGroupedExpression)
	p.registerPrefix(lexer.TokenIf, p.parseIfExpression)
	p.registerPrefix(lexer.TokenMatch, p.parseMatchExpression)
	p.registerPrefix(lexer.TokenFn, p.parseFunctionExpression)
	p.registerPrefix(lexer.TokenStruct, p.parseStructExpression)
	p.registerPrefix(lexer.TokenEnum, p.parseEnumExpression)

	// infix/binary operators
	p.registerInfix(lexer.TokenPlus, p.parseInfixExpression)
	p.registerInfix(lexer.TokenMinus, p.parseInfixExpression)
	p.registerInfix(lexer.TokenSlash, p.parseInfixExpression)
	p.registerInfix(lexer.TokenMultiply, p.parseInfixExpression)
	p.registerInfix(lexer.TokenModule, p.parseInfixExpression)
	p.registerInfix(lexer.TokenAssign, p.parseInfixExpression)
	p.registerInfix(lexer.TokenAnd, p.parseInfixExpression)
	p.registerInfix(lexer.TokenOr, p.parseInfixExpression)
	p.registerInfix(lexer.TokenEquals, p.parseInfixExpression)
	p.registerInfix(lexer.TokenNotEquals, p.parseInfixExpression)
	p.registerInfix(lexer.TokenLess, p.parseInfixExpression)
	p.registerInfix(lexer.TokenGreater, p.parseInfixExpression)
	p.registerInfix(lexer.TokenLessOrEqual, p.parseInfixExpression)
	p.registerInfix(lexer.TokenGreaterOrEqual, p.parseInfixExpression)
	p.registerInfix(lexer.TokenBraceOpen, p.parseCallExpression)
	p.registerInfix(lexer.TokenBracketOpen, p.parseIndexExpression)
	p.registerInfix(lexer.TokenCurlyBraceOpen, p.parseCurlyBraceOpen)
	p.registerInfix(lexer.TokenDot, p.parseMemberShipAccess)

	return &p
}

func (p *Parser) peekPrecedence() int {
	if p, ok := precedences[p.currentToken().Kind]; ok {
		return p
	}
	return LOWEST
}

func (p *Parser) nextToken() lexer.Token {
	if p.Pos >= len(p.Tokens) {
		return lexer.Token{LiteralToken: lexer.LiteralToken{Kind: lexer.TokenEOF}}
	}
	tok := p.Tokens[p.Pos]
	p.Pos++
	return tok
}

func (p *Parser) lookToken(move int) lexer.Token {
	peekPos := p.Pos + move
	if peekPos >= len(p.Tokens) {
		return lexer.Token{LiteralToken: lexer.LiteralToken{Kind: lexer.TokenEOF}}
	}
	return p.Tokens[peekPos]
}

// Returns the current lexer.token to process, if none, returns the EOF
func (p *Parser) currentToken() lexer.Token {
	if p.Pos >= len(p.Tokens) {
		return lexer.Token{LiteralToken: lexer.LiteralToken{Kind: lexer.TokenEOF}}
	}
	return p.Tokens[p.Pos]
}

func (p *Parser) expect(kinds []lexer.TokenKind) bool {
	tok := p.nextToken()
	if slices.Index(kinds, tok.Kind) == -1 {
		p.Errors = append(p.Errors, p.error(tok, fmt.Sprintf("ERROR: expected one of (%v), received %v", kinds, tok.Kind)))
		return false
	}

	return true
}

func (p *Parser) error(tok lexer.Token, msg string) error {
	switch tok.Kind {
	case lexer.TokenCurlyBraceOpen, lexer.TokenCurlyBraceClose, lexer.TokenColon:
		tok = p.Tokens[p.Pos-2]
		tok.Col = tok.Col + len(tok.Text)
	case lexer.TokenEOF:
		tok = p.Tokens[p.Pos-2]
		tok.Col = tok.Col + len(tok.Text) + 1
	default:
		if key, isMatching := lexer.Keywords[tok.Text]; isMatching && key != lexer.TokenBool {
			prev := p.Tokens[p.Pos-2]
			if tok.Row >= prev.Row {
				tok = prev
				tok.Col = tok.Col + len(tok.Text) + 1
			}
		}
	}

	errMsg := fmt.Sprintf("\033[1;90m%s:%d:%d:\033[0m\n\n", p.FilePath, tok.Row, tok.Col)

	// Build row set map
	rowSet := make(map[int][]lexer.Token)
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
	rowMap := make(map[int][]lexer.Token)
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
			if t.Kind == lexer.TokenString {
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
	for p.currentToken().Row <= tok.Row {
		p.nextToken()
	}
	return errors.New(errMsg)
}

func (p *Parser) registerPrefix(tokenType lexer.TokenKind, fn prefixParseFn) {
	p.prefixParseFns[tokenType] = fn
}
func (p *Parser) registerInfix(tokenType lexer.TokenKind, fn infixParseFn) {
	p.infixParseFns[tokenType] = fn
}

func (p *Parser) Parse() *ast.Program {
	ast := ast.Program{
		Statements: []ast.Statement{},
	}

	for p.currentToken().Kind != lexer.TokenEOF {
		stmt, err := p.parseStatement()
		if err != nil {
			p.Errors = append(p.Errors, err)
			return nil
		} else {
			ast.Statements = append(ast.Statements, stmt)
		}
	}

	return &ast
}

// TODO: better error handling and targeting

func (p *Parser) parseStatement() (ast.Statement, error) {
	stmtToken := p.currentToken() // Consume stmt
	switch stmtToken.Kind {
	case lexer.TokenLet, lexer.TokenConst:
		return p.parseVarDeclaration()
	case lexer.TokenReturn:
		return p.parseReturnStatement()
	case lexer.TokenImport:
		return p.parseImportStatement()
	case lexer.TokenWhile:
		return p.parseWhileStatement()
	case lexer.TokenFor:
		return p.parseForStatement()
	case lexer.TokenIdentifier:
		firstLookKind := p.lookToken(1).Kind
		// check after it if there is a colon and a {
		if firstLookKind == lexer.TokenColon && p.lookToken(2).Kind == lexer.TokenCurlyBraceOpen {
			return p.parseScope()
		}

		// for the bind operations, either :: or := or :
		if firstLookKind == lexer.TokenBind || firstLookKind == lexer.TokenWalrus || firstLookKind == lexer.TokenColon {
			return p.parseBindExpression()
		}
		return p.parseExpressionStatement()
	default:
		return p.parseExpressionStatement()
	}
}

func (p *Parser) parseVarDeclaration() (*ast.VarDeclaration, error) {
	stmt := &ast.VarDeclaration{Token: p.currentToken()}
	p.nextToken()
	identifier := p.parseIdentifier()
	if identifier == nil {
		return nil, p.Errors[len(p.Errors)-1]
	}

	stmt.Name = identifier.(*ast.Identifier)

	tok := p.nextToken()

	if tok.Kind != lexer.TokenAssign {
		return nil, p.error(tok, "ERROR: expected assign (=), got shit")
	}

	stmt.Value = p.parseExpression(LOWEST)

	return stmt, nil
}

func (p *Parser) parseReturnStatement() (*ast.ReturnStatement, error) {
	stmt := &ast.ReturnStatement{Token: p.currentToken()}
	p.nextToken()
	stmt.ReturnValue = p.parseExpression(LOWEST)
	return stmt, nil
}

func (p *Parser) parseImportStatement() (*ast.ImportStatement, error) {
	stmt := &ast.ImportStatement{Token: p.currentToken()}
	// skip import
	p.nextToken()

	// get the current after tok
	tok := p.currentToken()

	if tok.Kind != lexer.TokenString {
		return nil, p.error(tok, "ERROR: expected a string as module name, got shit")
	}

	stmt.ModuleName = p.parseStringLiteral().(*ast.StringLiteral)

	return stmt, nil
}

func (p *Parser) parseExpressionStatement() (*ast.ExpressionStatement, error) {
	stmt := &ast.ExpressionStatement{Token: p.currentToken()}

	expr := p.parseExpression(LOWEST)
	if expr == nil {
		return nil, fmt.Errorf("ERROR: on the ast.Expression stmt")
	}
	stmt.Expression = expr

	return stmt, nil
}

func (p *Parser) parseStructExpression() ast.Expression {
	expr := &ast.StructExpression{Token: p.currentToken()}
	p.nextToken()

	if !p.expect([]lexer.TokenKind{lexer.TokenCurlyBraceOpen}) {
		p.Errors = append(p.Errors, p.error(p.currentToken(), "ERROR: expected curl, got shit"))
		return nil
	}

	tok := p.currentToken()

	if tok.Kind == lexer.TokenBracketClose {
		p.nextToken()
		return &ast.StructExpression{
			Token:   expr.Token,
			Fields:  []*ast.Identifier{},
			Methods: []*ast.Method{},
		}
	}

	expr.Fields, expr.Methods = p.parseFields()

	return expr
}

func (p *Parser) parseFields() ([]*ast.Identifier, []*ast.Method) {
	fields := make([]*ast.Identifier, 0)
	methods := make([]*ast.Method, 0)

	field, ok := p.parseIdentifier().(*ast.Identifier)

	if !ok {
		p.Errors = append(p.Errors, p.error(p.lookToken(-1), "ERROR: expected an identifier, got shit"))
		return nil, nil
	}

	tok := p.nextToken()

	if tok.Kind == lexer.TokenColon {
		methods = append(methods, &ast.Method{
			Key:   field,
			Value: p.parseFunctionExpression().(*ast.FunctionExpression),
		})
	} else {
		fields = append(fields, field)
		p.Pos--
	}

	for p.currentToken().Kind == lexer.TokenComma {
		p.nextToken()
		field, ok := p.parseIdentifier().(*ast.Identifier)

		if !ok {
			p.Errors = append(p.Errors, p.error(p.lookToken(-1), "ERROR: expected an identifier, got shit"))
			return nil, nil
		}

		tok := p.nextToken()

		if tok.Kind == lexer.TokenColon {
			methods = append(methods, &ast.Method{
				Key:   field,
				Value: p.parseFunctionExpression().(*ast.FunctionExpression),
			})
		} else {
			fields = append(fields, field)
			p.Pos--
		}
	}

	if !p.expect([]lexer.TokenKind{lexer.TokenCurlyBraceClose}) {
		p.Errors = append(p.Errors, p.error(p.currentToken(), "ERROR: expected close curly brace ( } ), got shit"))
		return nil, nil
	}

	return fields, methods
}

func (p *Parser) parseEnumExpression() ast.Expression {
	expr := &ast.EnumExpression{Token: p.currentToken()}

	// consume the enum lexer.token
	p.nextToken()

	if !p.expect([]lexer.TokenKind{lexer.TokenCurlyBraceOpen}) {
		p.Errors = append(p.Errors, p.error(p.currentToken(), "ERROR: expected curl, got shit"))
		return nil
	}

	tok := p.currentToken()

	if tok.Kind == lexer.TokenBracketClose {
		p.nextToken()
		return &ast.EnumExpression{
			Token: expr.Token,
			Body:  []*ast.Identifier{},
		}
	}

	expr.Body = p.parseEnumFields()
	return expr

}

func (p *Parser) parseEnumFields() []*ast.Identifier {
	fields := make([]*ast.Identifier, 0)

	field, ok := p.parseIdentifier().(*ast.Identifier)

	if !ok {
		p.Errors = append(p.Errors, p.error(p.lookToken(-1), "ERROR: expected an identifier, got shit"))
		return nil
	}

	fields = append(fields, field)

	for p.currentToken().Kind == lexer.TokenComma {
		p.nextToken()
		field, ok := p.parseIdentifier().(*ast.Identifier)

		if !ok {
			p.Errors = append(p.Errors, p.error(p.lookToken(-1), "ERROR: expected an identifier, got shit"))
			return nil
		}

		fields = append(fields, field)
	}

	if !p.expect([]lexer.TokenKind{lexer.TokenCurlyBraceClose}) {
		p.Errors = append(p.Errors, p.error(p.currentToken(), "ERROR: expected close curly brace ( } ), got shit"))
		return nil
	}

	return fields
}

func (p *Parser) parseWhileStatement() (*ast.WhileStatement, error) {
	stmt := &ast.WhileStatement{Token: p.currentToken()}
	p.nextToken()

	stmt.Condition = p.parseExpression(ASSIGN)

	if !p.expect([]lexer.TokenKind{lexer.TokenCurlyBraceOpen}) {
		return nil, fmt.Errorf("ERROR: expected curly brace open ( { ), got shit")
	}

	stmt.Body = p.parseBlockStatement().(*ast.BlockStatement)
	return stmt, nil
}

func (p *Parser) parseForStatement() (*ast.ForStatement, error) {
	stmt := &ast.ForStatement{Token: p.currentToken()}
	p.nextToken()

	tok := p.currentToken()

	if tok.Kind != lexer.TokenIdentifier {
		return nil, p.error(tok, "ERROR: expected at least one identifier, got shit")
	}

	stmt.Identifiers = append(stmt.Identifiers, p.parseIdentifier().(*ast.Identifier))

	tok = p.nextToken()

	if tok.Kind == lexer.TokenComma {
		ident, ok := p.parseIdentifier().(*ast.Identifier)
		if !ok {
			return nil, p.error(tok, "ERROR: expected an identifier, got shit")
		}
		stmt.Identifiers = append(stmt.Identifiers, ident)
	} else {
		p.Pos--
	}

	tok = p.nextToken()
	if tok.Kind != lexer.TokenIn {
		return nil, p.error(tok, "ERROR: expected in, got shit")
	}

	stmt.Target = p.parseExpression(OR)

	if !p.expect([]lexer.TokenKind{lexer.TokenCurlyBraceOpen}) {
		return nil, p.error(p.currentToken(), "ERROR: expected curly brace open ( { ), got shit")
	}

	stmt.Body = p.parseBlockStatement().(*ast.BlockStatement)
	return stmt, nil
}

func (p *Parser) parseIdentifier() ast.Expression {
	tok := p.nextToken()

	if tok.Kind != lexer.TokenIdentifier {
		p.Errors = append(p.Errors, p.error(tok, "ERROR: expected identifier, got shit"))
		return nil
	}

	return &ast.Identifier{
		Token: tok,
		Value: tok.Text,
	}
}

func (p *Parser) parseIntLiteral() ast.Expression {
	tok := p.nextToken()

	num, err := strconv.ParseInt(tok.Text, 0, 64)
	if err != nil {
		return nil
	}
	return &ast.IntegerLiteral{
		Token: tok,
		Value: num,
	}
}

func (p *Parser) parseFloatLiteral() ast.Expression {
	tok := p.nextToken()
	num, err := strconv.ParseFloat(tok.Text, 64)
	if err != nil {
		return nil
	}
	return &ast.FloatLiteral{
		Token: tok,
		Value: num,
	}
}

func (p *Parser) parseStringLiteral() ast.Expression {
	tok := p.nextToken()
	return &ast.StringLiteral{
		Token: tok,
		Value: tok.Text,
	}
}

func (p *Parser) parseBooleanLiteral() ast.Expression {
	tok := p.nextToken()
	truth := tok.Text == "true"
	return &ast.BooleanLiteral{
		Token: tok,
		Value: truth,
	}
}

func (p *Parser) parseArrayLiteral() ast.Expression {
	prev := p.currentToken()

	if !p.expect([]lexer.TokenKind{lexer.TokenBracketOpen}) {
		p.Errors = append(p.Errors, p.error(prev, "ERROR: expected open bracket [, got shit"))
		return nil
	}

	elements := make([]ast.Expression, 0)

	tok := p.currentToken()

	if tok.Kind == lexer.TokenBracketClose {
		p.nextToken()
		return &ast.ArrayLiteral{
			Token:    prev,
			Elements: elements,
		}
	}

	elements = append(elements, p.parseExpression(LOWEST))

	for p.currentToken().Kind == lexer.TokenComma {
		p.nextToken()
		elements = append(elements, p.parseExpression(LOWEST))
	}

	if !p.expect([]lexer.TokenKind{lexer.TokenBracketClose}) {
		p.Errors = append(p.Errors, p.error(prev, "ERROR: expected close bracket ( ] ), got shit"))
		return nil
	}

	return &ast.ArrayLiteral{
		Token:    prev,
		Elements: elements,
	}
}

func (p *Parser) parseScope() (*ast.ScopeStatement, error) {
	stmt := &ast.ScopeStatement{Token: p.currentToken()}

	identifier := p.parseIdentifier()
	if identifier == nil {
		return nil, p.Errors[len(p.Errors)-1]
	}

	stmt.Name = identifier.(*ast.Identifier)
	tok := p.nextToken()

	if tok.Kind != lexer.TokenColon {
		return nil, p.error(tok, "ERROR: expected colon (:), got shit")
	}

	if !p.expect([]lexer.TokenKind{lexer.TokenCurlyBraceOpen}) {
		tok := p.currentToken()
		return nil, p.error(tok, "ERROR: expected ({), got shit")
	}

	stmt.Body = p.parseBlockStatement().(*ast.BlockStatement)
	return stmt, nil
}

func (p *Parser) parseMapLiteral() ast.Expression {
	prev := p.currentToken()

	if !p.expect([]lexer.TokenKind{lexer.TokenCurlyBraceOpen}) {
		p.Errors = append(p.Errors, p.error(prev, "ERROR: expected open curly- brace {, got shit"))
		return nil
	}

	pairs := make(map[ast.Expression]ast.Expression, 0)

	tok := p.currentToken()

	if tok.Kind == lexer.TokenCurlyBraceClose {
		p.nextToken()
		return &ast.MapLiteral{
			Token: prev,
			Pairs: pairs,
		}
	}

	key := p.parseExpression(LOWEST)

	tok = p.nextToken()

	if tok.Kind != lexer.TokenColon {
		p.Errors = append(p.Errors, p.error(tok, "ERROR: expected colon ( : ), got shit"))
		return nil
	}

	pairs[key] = p.parseExpression(LOWEST)

	for p.currentToken().Kind == lexer.TokenComma {
		p.nextToken()
		key := p.parseExpression(LOWEST)

		tok = p.nextToken()

		if tok.Kind != lexer.TokenColon {
			p.Errors = append(p.Errors, p.error(tok, "ERROR: expected colon ( : ), got shit"))
			return nil
		}

		pairs[key] = p.parseExpression(LOWEST)
	}

	if !p.expect([]lexer.TokenKind{lexer.TokenCurlyBraceClose}) {
		p.Errors = append(p.Errors, p.error(prev, "ERROR: expected close curly brace ( } ), got shit"))
		return nil
	}

	return &ast.MapLiteral{
		Token: prev,
		Pairs: pairs,
	}
}

func (p *Parser) parseGroupedExpression() ast.Expression {
	p.nextToken()
	exp := p.parseExpression(LOWEST)
	if !p.expect([]lexer.TokenKind{lexer.TokenBraceClose}) {
		return nil
	}
	return exp
}

func (p *Parser) parseIfExpression() ast.Expression {
	expr := &ast.IfExpression{Token: p.currentToken()}
	p.nextToken()

	// this is to prevent the launch of parse struct instance func
	p.internalFlags = append(p.internalFlags, "if-mode")
	expr.Condition = p.parseExpression(ASSIGN)
	p.internalFlags = slices.DeleteFunc(p.internalFlags, func(elem string) bool {
		return elem == "if-mode"
	})

	if !p.expect([]lexer.TokenKind{lexer.TokenCurlyBraceOpen}) {
		p.Errors = append(p.Errors, p.error(p.currentToken(), "ERROR: expected close curly brace ( } ), got shit"))
		return nil
	}

	expr.Consequence = p.parseBlockStatement().(*ast.BlockStatement)

	tok := p.nextToken()

	// check if there is an else stmt
	if tok.Kind == lexer.TokenElse {
		tok = p.currentToken()
		// support for else if
		if tok.Kind == lexer.TokenIf {
			expr.Alternative = p.parseIfExpression()
		} else {
			if !p.expect([]lexer.TokenKind{lexer.TokenCurlyBraceOpen}) {
				return nil
			}
			expr.Alternative = p.parseBlockStatement()
		}
	} else {
		p.Pos--
	}

	return expr
}

func (p *Parser) parseMatchExpression() ast.Expression {
	expr := &ast.MatchExpression{Token: p.currentToken()}
	// consume math keyword
	p.nextToken()

	expr.MatchKey = p.parseExpression(ASSIGN)

	if !p.expect([]lexer.TokenKind{lexer.TokenCurlyBraceOpen}) {
		p.Errors = append(p.Errors, p.error(p.currentToken(), "ERROR: expected close curly brace '{', got shit"))
		return nil
	}

	matchArms := make([]ast.MatchArm, 0)
	tok := p.currentToken()

	if tok.Kind == lexer.TokenCurlyBraceClose {
		p.nextToken()
		expr.Arms = matchArms
		return expr
	}

	pattern := p.parseExpression(LOWEST)

	tok = p.nextToken()

	if tok.Kind != lexer.TokenMatch {
		p.Errors = append(p.Errors, p.error(tok, "ERROR: expected colon ( : ), got shit"))
		return nil
	}

	if !p.expect([]lexer.TokenKind{lexer.TokenCurlyBraceOpen}) {
		p.Errors = append(p.Errors, p.error(p.currentToken(), "ERROR: expected close curly brace '{', got shit"))
		return nil
	}

	value := p.parseBlockStatement().(*ast.BlockStatement)

	matchArms = append(matchArms, ast.MatchArm{
		Token:   pattern.GetToken(),
		Pattern: pattern,
		Body:    value,
	})

	for p.currentToken().Kind == lexer.TokenComma {
		p.nextToken()

		patterCase := p.currentToken()

		pattern := p.parseExpression(LOWEST)
		tok = p.nextToken()

		if tok.Kind != lexer.TokenMatch {
			p.Errors = append(p.Errors, p.error(tok, "ERROR: expected colon ( : ), got shit"))
			return nil
		}

		if !p.expect([]lexer.TokenKind{lexer.TokenCurlyBraceOpen}) {
			p.Errors = append(p.Errors, p.error(p.currentToken(), "ERROR: expected close curly brace '{', got shit"))
			return nil
		}

		value := p.parseBlockStatement().(*ast.BlockStatement)

		if patterCase.Text == "_" {
			// default case
			expr.Default = &ast.MatchArm{
				Token:   pattern.GetToken(),
				Pattern: pattern,
				Body:    value,
			}
		} else {
			matchArms = append(matchArms, ast.MatchArm{
				Token:   pattern.GetToken(),
				Pattern: pattern,
				Body:    value,
			})
		}
	}
	expr.Arms = matchArms

	if !p.expect([]lexer.TokenKind{lexer.TokenCurlyBraceClose}) {
		p.Errors = append(p.Errors, p.error(p.currentToken(), "ERROR: expected close curly brace '}', got shit"))
		return nil
	}

	return expr
}

func (p *Parser) parseFunctionExpression() ast.Expression {
	expr := &ast.FunctionExpression{Token: p.currentToken()}
	p.nextToken()

	if !p.expect([]lexer.TokenKind{lexer.TokenBraceOpen}) {
		p.Errors = append(p.Errors, p.error(p.currentToken(), "ERROR: expected brace open '(' , got shit"))
		return nil
	}

	args := p.parseArguments()

	if args == nil {
		p.Errors = append(p.Errors, p.error(p.currentToken(), "ERROR: expected arguments, got shit"))
		return nil
	}

	expr.Args = args

	if !p.expect([]lexer.TokenKind{lexer.TokenCurlyBraceOpen}) {
		p.Errors = append(p.Errors, fmt.Errorf("ERROR: expected curly brace open ( { ), got shit"))
		return nil
	}

	body := p.parseBlockStatement().(*ast.BlockStatement)

	if body == nil {
		p.Errors = append(p.Errors, p.error(p.currentToken(), "ERROR: expected valid body, got shit"))
		return nil
	}

	expr.Body = body

	return expr
}

func (p *Parser) parseArguments() []*ast.Identifier {
	args := make([]*ast.Identifier, 0)

	if p.currentToken().Kind == lexer.TokenBraceClose {
		p.nextToken()
		return args
	}

	ident := &ast.Identifier{
		Token: p.currentToken(),
		Value: p.currentToken().Text,
	}

	args = append(args, ident)
	p.nextToken()

	for p.currentToken().Kind == lexer.TokenComma {
		p.nextToken()
		ident := &ast.Identifier{
			Token: p.currentToken(),
			Value: p.currentToken().Text,
		}

		args = append(args, ident)
		p.nextToken()
	}

	if !p.expect([]lexer.TokenKind{lexer.TokenBraceClose}) {
		return nil
	}

	return args
}

func (p *Parser) parseBlockStatement() ast.Expression {
	block := ast.BlockStatement{Token: p.currentToken()}
	block.Body = make([]ast.Statement, 0)

	for p.currentToken().Kind != lexer.TokenCurlyBraceClose && p.currentToken().Kind != lexer.TokenEOF {
		// parse body expressions and statements
		stmt, err := p.parseStatement()

		if err != nil {
			p.Errors = append(p.Errors, err)
		} else {
			block.Body = append(block.Body, stmt)
		}
	}

	if !p.expect([]lexer.TokenKind{lexer.TokenCurlyBraceClose}) {
		return nil
	}

	return &block
}

func (p *Parser) parseCallExpression(left ast.Expression) ast.Expression {
	switch left.(type) {
	case *ast.Identifier:
	default:
		p.Errors = append(p.Errors, p.error(p.currentToken(), "ERROR: only call are allowed, bounding function into a variable ain't allowed"))
		return nil
	}

	exp := ast.CallExpression{Token: left.GetToken(), Function: *(left.(*ast.Identifier))}

	exp.Args = p.parseCallArguments()

	return &exp
}

func (p *Parser) parseCallArguments() []ast.Expression {
	args := make([]ast.Expression, 0)
	if !p.expect([]lexer.TokenKind{lexer.TokenBraceOpen}) {
		return nil
	}

	if p.currentToken().Kind == lexer.TokenBraceClose {
		p.nextToken()
		return args
	}

	args = append(args, p.parseExpression(LOWEST))

	for p.currentToken().Kind == lexer.TokenComma {
		p.nextToken()
		args = append(args, p.parseExpression(LOWEST))
	}

	if !p.expect([]lexer.TokenKind{lexer.TokenBraceClose}) {
		return nil
	}

	return args
}

func (p *Parser) parseIndexExpression(left ast.Expression) ast.Expression {
	// TODO: if left expr is nil return an error
	exp := &ast.IndexExpression{Token: left.GetToken(), Left: left}

	p.nextToken()

	exp.Index = p.parseExpression(LOWEST)

	if !p.expect([]lexer.TokenKind{lexer.TokenBracketClose}) {
		return nil
	}

	return exp
}

func (p *Parser) parseCurlyBraceOpen(left ast.Expression) ast.Expression {
	if slices.Index(p.internalFlags, "if-mode") != -1 {
		return p.parseBlockStatement()
	} else {
		return p.parseStructInstanceExpression(left)
	}
}

func (p *Parser) parseStructInstanceExpression(left ast.Expression) ast.Expression {
	expr := &ast.StructInstanceExpression{Token: left.GetToken(), Left: left}
	expr.Body = p.parseFieldValues()

	return expr
}

func (p *Parser) parseFieldValues() []ast.FieldInstance {
	fields := make([]ast.FieldInstance, 0)
	p.nextToken()

	identifier, ok := p.parseIdentifier().(*ast.Identifier)

	if !ok {
		p.Errors = append(p.Errors, p.error(p.currentToken(), "ERROR: expected an identifier, got shit"))
		return []ast.FieldInstance{}
	}

	tok := p.nextToken()

	if tok.Kind != lexer.TokenColon {
		return []ast.FieldInstance{}
	}

	value := p.parseExpression(LOWEST)

	fields = append(fields, ast.FieldInstance{
		Key:   identifier,
		Value: value,
	})

	for p.currentToken().Kind == lexer.TokenComma {
		p.nextToken()
		identifier, ok := p.parseIdentifier().(*ast.Identifier)

		if !ok {
			return fields
		}

		tok := p.nextToken()

		if tok.Kind != lexer.TokenColon {
			return fields
		}

		value := p.parseExpression(LOWEST)

		fields = append(fields, ast.FieldInstance{
			Key:   identifier,
			Value: value,
		})
	}

	tok = p.nextToken()

	if tok.Kind != lexer.TokenCurlyBraceClose {
		return fields
	}

	return fields
}

func (p *Parser) parseMemberShipAccess(left ast.Expression) ast.Expression {
	expr := &ast.MemberShipExpression{Token: left.GetToken(), Object: left}

	if !p.expect([]lexer.TokenKind{lexer.TokenDot}) {
		return nil
	}

	expr.Property = p.parseExpression(ASSIGN)

	return expr
}

func (p *Parser) parseBindExpression() (ast.Statement, error) {
	stmt := &ast.VarDeclaration{Token: lexer.Token{
		LiteralToken: lexer.LiteralToken{
			Text: "let",
			Kind: lexer.TokenLet,
		},
		Col: p.currentToken().Col,
		Row: p.currentToken().Row,
	}}

	identifier, ok := p.parseIdentifier().(*ast.Identifier)

	if !ok {
		return nil, p.error(identifier.GetToken(), "ERROR: expected an identifier, got shit")
	}

	stmt.Name = identifier

	tok := p.nextToken()

	switch tok.Kind {
	case lexer.TokenBind:
		stmt.Token = lexer.Token{
			LiteralToken: lexer.LiteralToken{
				Text: "const",
				Kind: lexer.TokenConst,
			},
		}
		// fall through
	case lexer.TokenWalrus:
		// fall through
	default:
		return nil, p.error(tok, "ERROR: expected (:= or ::) operators, got shit")
	}

	value := p.parseExpression(LOWEST)

	if value == nil {
		return nil, p.error(tok, "ERROR: expected an ast.Expression, got nil value")
	}

	stmt.Value = value

	return stmt, nil
}

func (p *Parser) parsePrefixExpression() ast.Expression {
	tok := p.nextToken()

	if _, ok := lexer.UnaryOperators[tok.Kind]; !ok {
		p.Errors = append(p.Errors, p.error(tok, "ERROR: expected a unary operator (! | -), got shut"))
		return nil
	}

	right := p.parseExpression(PREFIX)

	return &ast.UnaryExpression{
		Token:    tok,
		Operator: tok.Text,
		Right:    right,
	}
}

func (p *Parser) parseInfixExpression(left ast.Expression) ast.Expression {
	tok := p.currentToken()

	if _, ok := lexer.BinOperators[tok.Kind]; !ok {
		p.Errors = append(p.Errors, p.error(tok, "ERROR: expected a binary operator (== | > | < | ...), got shut"))
		return nil
	}

	precedence := p.peekPrecedence()
	p.nextToken()
	right := p.parseExpression(precedence)

	return &ast.BinaryExpression{
		Token:    tok,
		Operator: tok.Text,
		Left:     left,
		Right:    right,
	}
}

func (p *Parser) parseExpression(precedence int) ast.Expression {
	prefix := p.prefixParseFns[p.currentToken().Kind]

	if prefix == nil {
		p.Errors = append(p.Errors, p.error(p.currentToken(), "ERROR: ain't an ast.Expression, it is a statement"))
		return nil
	}

	leftExp := prefix()

	cur := p.currentToken()

	for p.currentToken().Row <= cur.Row && precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.currentToken().Kind]
		if infix == nil {
			return leftExp
		}
		leftExp = infix(leftExp)
	}

	return leftExp
}
