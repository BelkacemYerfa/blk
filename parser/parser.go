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

var precedences = map[TokenKind]int{
	TokenCurlyBraceOpen: ASSIGN,
	TokenAssign:         ASSIGN,
	TokenBind:           ASSIGN,
	TokenWalrus:         ASSIGN,
	TokenOr:             OR,
	TokenAnd:            AND,
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
	TokenModule:         PRODUCT,
	TokenBraceOpen:      CALL,
	TokenBracketOpen:    INDEX,
	TokenDot:            STRUCT,
}

func NewParser(tokens []Token, filepath string) *Parser {
	p := Parser{
		Tokens:         tokens,
		FilePath:       filepath,
		Errors:         []error{},
		prefixParseFns: make(map[TokenKind]prefixParseFn),
		infixParseFns:  make(map[TokenKind]infixParseFn),
		Pos:            0,
		internalFlags:  []string{},
	}

	// prefix/unary operators
	p.registerPrefix(TokenIdentifier, p.parseIdentifier)
	p.registerPrefix(TokenArray, p.ParseType)
	p.registerPrefix(TokenMap, p.ParseType)
	p.registerPrefix(TokenInt, p.parseIntLiteral)
	p.registerPrefix(TokenFloat, p.parseFloatLiteral)
	p.registerPrefix(TokenString, p.parseStringLiteral)
	p.registerPrefix(TokenBracketOpen, p.parseArrayLiteral)
	p.registerPrefix(TokenCurlyBraceOpen, p.parseMapLiteral)
	p.registerPrefix(TokenExclamation, p.parsePrefixExpression)
	p.registerPrefix(TokenMinus, p.parsePrefixExpression)
	p.registerPrefix(TokenBool, p.parseBooleanLiteral)
	p.registerPrefix(TokenBraceOpen, p.parseGroupedExpression)
	p.registerPrefix(TokenIf, p.parseIfExpression)
	p.registerPrefix(TokenMatch, p.parseMatchExpression)
	p.registerPrefix(TokenFn, p.parseFunctionExpression)
	p.registerPrefix(TokenStruct, p.parseStructExpression)
	p.registerPrefix(TokenEnum, p.parseEnumExpression)

	// infix/binary operators
	p.registerInfix(TokenPlus, p.parseInfixExpression)
	p.registerInfix(TokenMinus, p.parseInfixExpression)
	p.registerInfix(TokenSlash, p.parseInfixExpression)
	p.registerInfix(TokenMultiply, p.parseInfixExpression)
	p.registerInfix(TokenModule, p.parseInfixExpression)
	p.registerInfix(TokenAssign, p.parseInfixExpression)
	p.registerInfix(TokenAnd, p.parseInfixExpression)
	p.registerInfix(TokenOr, p.parseInfixExpression)
	p.registerInfix(TokenEquals, p.parseInfixExpression)
	p.registerInfix(TokenNotEquals, p.parseInfixExpression)
	p.registerInfix(TokenLess, p.parseInfixExpression)
	p.registerInfix(TokenGreater, p.parseInfixExpression)
	p.registerInfix(TokenLessOrEqual, p.parseInfixExpression)
	p.registerInfix(TokenGreaterOrEqual, p.parseInfixExpression)
	p.registerInfix(TokenBraceOpen, p.parseCallExpression)
	p.registerInfix(TokenBracketOpen, p.parseIndexExpression)
	p.registerInfix(TokenCurlyBraceOpen, p.parseCurlyBraceOpen)
	p.registerInfix(TokenDot, p.parseMemberShipAccess)

	return &p
}

func (p *Parser) peekPrecedence() int {
	if p, ok := precedences[p.currentToken().Kind]; ok {
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

func (p *Parser) lookToken(move int) Token {
	peekPos := p.Pos + move
	if peekPos >= len(p.Tokens) {
		return Token{LiteralToken: LiteralToken{Kind: TokenEOF}}
	}
	return p.Tokens[peekPos]
}

// Returns the current token to process, if none, returns the EOF
func (p *Parser) currentToken() Token {
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
	for p.currentToken().Row <= tok.Row {
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

	for p.currentToken().Kind != TokenEOF {
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

func (p *Parser) parseStatement() (Statement, error) {
	stmtToken := p.currentToken() // Consume stmt
	switch stmtToken.Kind {
	case TokenLet:
		return p.parseVarDeclaration()
	case TokenReturn:
		return p.parseReturnStatement()
	case TokenImport:
		return p.parseImportStatement()
	case TokenWhile:
		return p.parseWhileStatement()
	case TokenFor:
		return p.parseForStatement()
	case TokenIdentifier:
		firstLookKind := p.lookToken(1).Kind
		// check after it if there is a colon and a {
		if firstLookKind == TokenColon && p.lookToken(2).Kind == TokenCurlyBraceOpen {
			return p.parseScope()
		}

		// for the bind operations, either :: or := or :
		if firstLookKind == TokenBind || firstLookKind == TokenWalrus || firstLookKind == TokenColon {
			return p.parseBindExpression()
		}
		return p.parseExpressionStatement()
	default:
		return p.parseExpressionStatement()
	}
}

func (p *Parser) parseVarDeclaration() (*VarDeclaration, error) {
	stmt := &VarDeclaration{Token: p.currentToken()}
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

	stmt.ExplicitType = p.ParseType()

	tok = p.currentToken()

	if tok.Kind != TokenAssign {
		return nil, p.error(tok, "ERROR: expected assign (=), got shit")
	}

	p.nextToken()
	stmt.Value = p.parseExpression(LOWEST)

	return stmt, nil
}

func (p *Parser) parseReturnStatement() (*ReturnStatement, error) {
	stmt := &ReturnStatement{Token: p.currentToken()}
	p.nextToken()
	stmt.ReturnValue = p.parseExpression(LOWEST)
	return stmt, nil
}

func (p *Parser) parseImportStatement() (*ImportStatement, error) {
	stmt := &ImportStatement{Token: p.currentToken()}
	// skip import
	p.nextToken()

	// get the current after tok
	tok := p.currentToken()

	if tok.Kind != TokenString {
		return nil, p.error(tok, "ERROR: expected a string as module name, got shit")
	}

	stmt.ModuleName = p.parseStringLiteral().(*StringLiteral)

	return stmt, nil
}

func (p *Parser) parseExpressionStatement() (*ExpressionStatement, error) {
	stmt := &ExpressionStatement{Token: p.currentToken()}

	expr := p.parseExpression(LOWEST)
	if expr == nil {
		return nil, fmt.Errorf("ERROR: on the expression stmt")
	}
	stmt.Expression = expr

	return stmt, nil
}

func (p *Parser) parseStructExpression() Expression {
	expr := &StructExpression{Token: p.currentToken()}
	p.nextToken()

	if !p.expect([]TokenKind{TokenCurlyBraceOpen}) {
		p.Errors = append(p.Errors, p.error(p.currentToken(), "ERROR: expected curl, got shit"))
		return nil
	}

	tok := p.currentToken()

	if tok.Kind == TokenBracketClose {
		p.nextToken()
		return &StructExpression{
			Token: expr.Token,
			Body:  []Field{},
		}
	}

	expr.Body = p.parseFields()

	return expr
}

func (p *Parser) parseFields() []Field {
	fields := make([]Field, 0)

	field, ok := p.parseIdentifier().(*Identifier)

	if !ok {
		p.Errors = append(p.Errors, p.error(p.lookToken(-1), "ERROR: expected an identifier, got shit"))
		return nil
	}

	tok := p.nextToken()

	if tok.Kind != TokenColon {
		p.Errors = append(p.Errors, p.error(tok, "ERROR: expected colon ( : ), got shit"))
		return nil
	}

	fieldValue := p.parseExpression(LOWEST)

	fields = append(fields, Field{
		Key:   field,
		Value: fieldValue,
	})

	for p.currentToken().Kind == TokenComma {
		p.nextToken()
		field, ok := p.parseIdentifier().(*Identifier)

		if !ok {
			p.Errors = append(p.Errors, p.error(p.lookToken(-1), "ERROR: expected an identifier, got shit"))
			return nil
		}

		tok := p.nextToken()

		if tok.Kind != TokenColon {
			p.Errors = append(p.Errors, p.error(tok, "ERROR: expected colon ( : ), got shit"))
			return nil
		}

		fieldValue := p.parseExpression(LOWEST)

		fields = append(fields, Field{
			Key:   field,
			Value: fieldValue,
		})
	}

	if !p.expect([]TokenKind{TokenCurlyBraceClose}) {
		p.Errors = append(p.Errors, p.error(p.currentToken(), "ERROR: expected close curly brace ( } ), got shit"))
		return nil
	}

	return fields
}

func (p *Parser) parseEnumExpression() Expression {
	expr := &EnumExpression{Token: p.currentToken()}

	// consume the enum token
	p.nextToken()

	if !p.expect([]TokenKind{TokenCurlyBraceOpen}) {
		p.Errors = append(p.Errors, p.error(p.currentToken(), "ERROR: expected curl, got shit"))
		return nil
	}

	tok := p.currentToken()

	if tok.Kind == TokenBracketClose {
		p.nextToken()
		return &EnumExpression{
			Token: expr.Token,
			Body:  []*Identifier{},
		}
	}

	expr.Body = p.parseEnumFields()
	return expr

}

func (p *Parser) parseEnumFields() []*Identifier {
	fields := make([]*Identifier, 0)

	field, ok := p.parseIdentifier().(*Identifier)

	if !ok {
		p.Errors = append(p.Errors, p.error(p.lookToken(-1), "ERROR: expected an identifier, got shit"))
		return nil
	}

	fields = append(fields, field)

	for p.currentToken().Kind == TokenComma {
		p.nextToken()
		field, ok := p.parseIdentifier().(*Identifier)

		if !ok {
			p.Errors = append(p.Errors, p.error(p.lookToken(-1), "ERROR: expected an identifier, got shit"))
			return nil
		}

		fields = append(fields, field)
	}

	if !p.expect([]TokenKind{TokenCurlyBraceClose}) {
		p.Errors = append(p.Errors, p.error(p.currentToken(), "ERROR: expected close curly brace ( } ), got shit"))
		return nil
	}

	return fields
}

func (p *Parser) parseWhileStatement() (*WhileStatement, error) {
	stmt := &WhileStatement{Token: p.currentToken()}
	p.nextToken()

	stmt.Condition = p.parseExpression(ASSIGN)

	if !p.expect([]TokenKind{TokenCurlyBraceOpen}) {
		return nil, fmt.Errorf("ERROR: expected curly brace open ( { ), got shit")
	}

	stmt.Body = p.parseBlockStatement().(*BlockStatement)
	return stmt, nil
}

func (p *Parser) parseForStatement() (*ForStatement, error) {
	stmt := &ForStatement{Token: p.currentToken()}
	p.nextToken()

	tok := p.currentToken()

	if tok.Kind != TokenIdentifier {
		return nil, p.error(tok, "ERROR: expected at least one identifier, got shit")
	}

	stmt.Identifiers = append(stmt.Identifiers, p.parseIdentifier().(*Identifier))

	tok = p.nextToken()

	if tok.Kind == TokenComma {
		ident, ok := p.parseIdentifier().(*Identifier)
		if !ok {
			return nil, p.error(tok, "ERROR: expected an identifier, got shit")
		}
		stmt.Identifiers = append(stmt.Identifiers, ident)
	} else {
		p.Pos--
	}

	tok = p.nextToken()
	if tok.Kind != TokenIn {
		return nil, p.error(tok, "ERROR: expected in, got shit")
	}

	stmt.Target = p.parseExpression(OR)

	if !p.expect([]TokenKind{TokenCurlyBraceOpen}) {
		return nil, p.error(p.currentToken(), "ERROR: expected curly brace open ( { ), got shit")
	}

	stmt.Body = p.parseBlockStatement().(*BlockStatement)
	return stmt, nil
}

func (p *Parser) ParseType() Expression {
	nodeType := &NodeType{Token: p.currentToken()}

	tok := p.nextToken()

	switch tok.Kind {
	case TokenIdentifier:
		nodeType.Type = p.typeMapper(tok.Text)
	case TokenArray:
		tok = p.nextToken() // consume (

		if p.currentToken().Kind != TokenArray {
			childType := p.ParseType()
			return &NodeType{
				Token:     p.currentToken(),
				Type:      "array",
				ChildType: childType.(*NodeType),
			}
		}

		for p.currentToken().Kind == TokenArray {
			childType := p.ParseType()
			return &NodeType{
				Token:     p.currentToken(),
				Type:      "array",
				ChildType: childType.(*NodeType),
			}
		}

	case TokenMap:
		tok = p.nextToken() // consume (

		if p.currentToken().Kind != TokenMap {
			keyType := p.ParseType()
			p.nextToken()
			valueType := p.ParseType()
			return &MapType{
				Token: p.currentToken(),
				Type:  "map",
				Left:  keyType.(Type),
				Right: valueType.(Type),
			}
		}

		for p.currentToken().Kind == TokenMap {
			keyType := p.ParseType()
			p.nextToken()
			valueType := p.ParseType()
			return &MapType{
				Token: p.currentToken(),
				Type:  "map",
				Left:  keyType.(Type),
				Right: valueType.(Type),
			}
		}

	case TokenBracketOpen:
		tok = p.currentToken()
		p.nextToken()
		p.nextToken()
		childType := p.ParseType()
		return &NodeType{
			Token:     p.currentToken(),
			Type:      "array",
			ChildType: childType.(*NodeType),
			Size:      tok.Text,
		}
	default:
		p.Pos--
		// if there is no type, empty, we return what we already have in the node type
		return nodeType
	}

	if p.nextToken().Kind == TokenBraceClose {
		for p.currentToken().Kind == TokenBraceClose {
			p.nextToken()
		}
	} else {
		p.Pos--
	}

	return nodeType
}

func (p *Parser) typeMapper(typ string) TYPE {
	if mappedType, isMatching := AtomicTypes[typ]; isMatching {
		return mappedType
	} else {
		return typ
	}
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
	prev := p.currentToken()

	if !p.expect([]TokenKind{TokenBracketOpen}) {
		p.Errors = append(p.Errors, p.error(prev, "ERROR: expected open bracket [, got shit"))
		return nil
	}

	elements := make([]Expression, 0)

	tok := p.currentToken()

	if tok.Kind == TokenBracketClose {
		p.nextToken()
		return &ArrayLiteral{
			Token:    prev,
			Elements: elements,
		}
	}

	elements = append(elements, p.parseExpression(LOWEST))

	for p.currentToken().Kind == TokenComma {
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

func (p *Parser) parseScope() (*ScopeStatement, error) {
	stmt := &ScopeStatement{Token: p.currentToken()}

	identifier := p.parseIdentifier()
	if identifier == nil {
		return nil, p.Errors[len(p.Errors)-1]
	}

	stmt.Name = identifier.(*Identifier)
	tok := p.nextToken()

	if tok.Kind != TokenColon {
		return nil, p.error(tok, "ERROR: expected colon (:), got shit")
	}

	if !p.expect([]TokenKind{TokenCurlyBraceOpen}) {
		tok := p.currentToken()
		return nil, p.error(tok, "ERROR: expected ({), got shit")
	}

	stmt.Body = p.parseBlockStatement().(*BlockStatement)
	return stmt, nil
}

func (p *Parser) parseMapLiteral() Expression {
	prev := p.currentToken()

	if !p.expect([]TokenKind{TokenCurlyBraceOpen}) {
		p.Errors = append(p.Errors, p.error(prev, "ERROR: expected open curly- brace {, got shit"))
		return nil
	}

	pairs := make(map[Expression]Expression, 0)

	tok := p.currentToken()

	if tok.Kind == TokenCurlyBraceClose {
		p.nextToken()
		return &MapLiteral{
			Token: prev,
			Pairs: pairs,
		}
	}

	key := p.parseExpression(LOWEST)

	tok = p.nextToken()

	if tok.Kind != TokenColon {
		p.Errors = append(p.Errors, p.error(tok, "ERROR: expected colon ( : ), got shit"))
		return nil
	}

	pairs[key] = p.parseExpression(LOWEST)

	for p.currentToken().Kind == TokenComma {
		p.nextToken()
		key := p.parseExpression(LOWEST)

		tok = p.nextToken()

		if tok.Kind != TokenColon {
			p.Errors = append(p.Errors, p.error(tok, "ERROR: expected colon ( : ), got shit"))
			return nil
		}

		pairs[key] = p.parseExpression(LOWEST)
	}

	if !p.expect([]TokenKind{TokenCurlyBraceClose}) {
		p.Errors = append(p.Errors, p.error(prev, "ERROR: expected close curly brace ( } ), got shit"))
		return nil
	}

	return &MapLiteral{
		Token: prev,
		Pairs: pairs,
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
	expr := &IfExpression{Token: p.currentToken()}
	p.nextToken()

	// this is to prevent the launch of parse struct instance func
	p.internalFlags = append(p.internalFlags, "if-mode")
	expr.Condition = p.parseExpression(ASSIGN)
	p.internalFlags = slices.DeleteFunc(p.internalFlags, func(elem string) bool {
		return elem == "if-mode"
	})

	if !p.expect([]TokenKind{TokenCurlyBraceOpen}) {
		p.Errors = append(p.Errors, p.error(p.currentToken(), "ERROR: expected close curly brace ( } ), got shit"))
		return nil
	}

	expr.Consequence = p.parseBlockStatement().(*BlockStatement)

	tok := p.nextToken()

	// check if there is an else stmt
	if tok.Kind == TokenElse {
		tok = p.currentToken()
		// support for else if
		if tok.Kind == TokenIf {
			expr.Alternative = p.parseIfExpression()
		} else {
			if !p.expect([]TokenKind{TokenCurlyBraceOpen}) {
				return nil
			}
			expr.Alternative = p.parseBlockStatement()
		}
	} else {
		p.Pos--
	}

	return expr
}

func (p *Parser) parseMatchExpression() Expression {
	expr := &MatchExpression{Token: p.currentToken()}
	// consume math keyword
	p.nextToken()

	expr.MatchKey = p.parseExpression(ASSIGN)

	if !p.expect([]TokenKind{TokenCurlyBraceOpen}) {
		p.Errors = append(p.Errors, p.error(p.currentToken(), "ERROR: expected close curly brace '{', got shit"))
		return nil
	}

	matchArms := make([]MatchArm, 0)
	tok := p.currentToken()

	if tok.Kind == TokenCurlyBraceClose {
		p.nextToken()
		expr.Arms = matchArms
		return expr
	}

	pattern := p.parseExpression(LOWEST)

	tok = p.nextToken()

	if tok.Kind != TokenMatch {
		p.Errors = append(p.Errors, p.error(tok, "ERROR: expected colon ( : ), got shit"))
		return nil
	}

	if !p.expect([]TokenKind{TokenCurlyBraceOpen}) {
		p.Errors = append(p.Errors, p.error(p.currentToken(), "ERROR: expected close curly brace '{', got shit"))
		return nil
	}

	value := p.parseBlockStatement().(*BlockStatement)

	matchArms = append(matchArms, MatchArm{
		Token:   pattern.GetToken(),
		Pattern: pattern,
		Body:    value,
	})

	for p.currentToken().Kind == TokenComma {
		p.nextToken()

		patterCase := p.currentToken()

		pattern := p.parseExpression(LOWEST)
		tok = p.nextToken()

		if tok.Kind != TokenMatch {
			p.Errors = append(p.Errors, p.error(tok, "ERROR: expected colon ( : ), got shit"))
			return nil
		}

		if !p.expect([]TokenKind{TokenCurlyBraceOpen}) {
			p.Errors = append(p.Errors, p.error(p.currentToken(), "ERROR: expected close curly brace '{', got shit"))
			return nil
		}

		value := p.parseBlockStatement().(*BlockStatement)

		if patterCase.Text == "_" {
			// default case
			expr.Default = &MatchArm{
				Token:   pattern.GetToken(),
				Pattern: pattern,
				Body:    value,
			}
		} else {
			matchArms = append(matchArms, MatchArm{
				Token:   pattern.GetToken(),
				Pattern: pattern,
				Body:    value,
			})
		}
	}
	expr.Arms = matchArms

	if !p.expect([]TokenKind{TokenCurlyBraceClose}) {
		p.Errors = append(p.Errors, p.error(p.currentToken(), "ERROR: expected close curly brace '}', got shit"))
		return nil
	}

	return expr
}

func (p *Parser) parseFunctionExpression() Expression {
	expr := &FunctionExpression{Token: p.currentToken()}
	p.nextToken()

	if !p.expect([]TokenKind{TokenBraceOpen}) {
		p.Errors = append(p.Errors, p.error(p.currentToken(), "ERROR: expected brace open '(' , got shit"))
		return nil
	}

	args := p.parseArguments()

	if args == nil {
		p.Errors = append(p.Errors, p.error(p.currentToken(), "ERROR: expected arguments, got shit"))
		return nil
	}

	expr.Args = args

	returnType := p.ParseType()
	expr.ReturnType = returnType

	if !p.expect([]TokenKind{TokenCurlyBraceOpen}) {
		p.Errors = append(p.Errors, fmt.Errorf("ERROR: expected curly brace open ( { ), got shit"))
		return nil
	}

	body := p.parseBlockStatement().(*BlockStatement)

	if body == nil {
		p.Errors = append(p.Errors, p.error(p.currentToken(), "ERROR: expected valid body, got shit"))
		return nil
	}

	expr.Body = body

	return expr
}

func (p *Parser) parseArguments() []*ArgExpression {
	args := make([]*ArgExpression, 0)

	if p.currentToken().Kind == TokenBraceClose {
		p.nextToken()
		return args
	}
	ident := &Identifier{
		Token: p.currentToken(),
		Value: p.currentToken().Text,
	}

	p.nextToken()
	if !p.expect([]TokenKind{TokenColon}) {
		return nil
	}

	if !p.expect([]TokenKind{TokenIdentifier}) {
		return nil
	}

	p.Pos--
	args = append(args, &ArgExpression{
		Identifier: ident,
		Type:       p.ParseType(),
	})

	for p.currentToken().Kind == TokenComma {
		p.nextToken()
		ident := &Identifier{
			Token: p.currentToken(),
			Value: p.currentToken().Text,
		}

		p.nextToken()
		if !p.expect([]TokenKind{TokenColon}) {
			return nil
		}

		if !p.expect([]TokenKind{TokenIdentifier}) {
			return nil
		}
		p.Pos--
		args = append(args, &ArgExpression{
			Identifier: ident,
			Type:       p.ParseType(),
		})
	}

	p.Pos--
	if !p.expect([]TokenKind{TokenBraceClose}) {
		return nil
	}

	return args
}

func (p *Parser) parseBlockStatement() Expression {
	block := BlockStatement{Token: p.currentToken()}
	block.Body = make([]Statement, 0)

	for p.currentToken().Kind != TokenCurlyBraceClose && p.currentToken().Kind != TokenEOF {
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
		p.Errors = append(p.Errors, p.error(p.currentToken(), "ERROR: only call are allowed, bounding function into a variable ain't allowed"))
		return nil
	}

	exp := CallExpression{Token: left.GetToken(), Function: *(left.(*Identifier))}

	exp.Args = p.parseCallArguments()

	return &exp
}

func (p *Parser) parseCallArguments() []Expression {
	args := make([]Expression, 0)
	if !p.expect([]TokenKind{TokenBraceOpen}) {
		return nil
	}

	if p.currentToken().Kind == TokenBraceClose {
		p.nextToken()
		return args
	}

	args = append(args, p.parseExpression(LOWEST))

	for p.currentToken().Kind == TokenComma {
		p.nextToken()
		args = append(args, p.parseExpression(LOWEST))
	}

	if !p.expect([]TokenKind{TokenBraceClose}) {
		return nil
	}

	return args
}

func (p *Parser) parseIndexExpression(left Expression) Expression {
	// TODO: if left expr is nil return an error
	exp := &IndexExpression{Token: left.GetToken(), Left: left}

	p.nextToken()

	exp.Index = p.parseExpression(LOWEST)

	if !p.expect([]TokenKind{TokenBracketClose}) {
		return nil
	}

	return exp
}

func (p *Parser) parseCurlyBraceOpen(left Expression) Expression {
	if slices.Index(p.internalFlags, "if-mode") != -1 {
		return p.parseBlockStatement()
	} else {
		return p.parseStructInstanceExpression(left)
	}
}

func (p *Parser) parseStructInstanceExpression(left Expression) Expression {
	expr := &StructInstanceExpression{Token: left.GetToken(), Left: left}
	expr.Body = p.parseFieldValues()

	return expr
}

func (p *Parser) parseFieldValues() []FieldInstance {
	fields := make([]FieldInstance, 0)
	p.nextToken()

	identifier, ok := p.parseIdentifier().(*Identifier)

	if !ok {
		p.Errors = append(p.Errors, p.error(p.currentToken(), "ERROR: expected an identifier, got shit"))
		return []FieldInstance{}
	}

	tok := p.nextToken()

	if tok.Kind != TokenColon {
		return []FieldInstance{}
	}

	value := p.parseExpression(LOWEST)

	fields = append(fields, FieldInstance{
		Key:   identifier,
		Value: value,
	})

	for p.currentToken().Kind == TokenComma {
		p.nextToken()
		identifier, ok := p.parseIdentifier().(*Identifier)

		if !ok {
			return fields
		}

		tok := p.nextToken()

		if tok.Kind != TokenColon {
			return fields
		}

		value := p.parseExpression(LOWEST)

		fields = append(fields, FieldInstance{
			Key:   identifier,
			Value: value,
		})
	}

	tok = p.nextToken()

	if tok.Kind != TokenCurlyBraceClose {
		return fields
	}

	return fields
}

func (p *Parser) parseMemberShipAccess(left Expression) Expression {
	expr := &MemberShipExpression{Token: left.GetToken(), Object: left}

	if !p.expect([]TokenKind{TokenDot}) {
		return nil
	}

	expr.Property = p.parseExpression(ASSIGN)

	return expr
}

func (p *Parser) parseBindExpression() (Statement, error) {
	stmt := &VarDeclaration{Token: p.currentToken()}

	identifier, ok := p.parseIdentifier().(*Identifier)

	if !ok {
		return nil, p.error(identifier.GetToken(), "ERROR: expected an identifier, got shit")
	}

	stmt.Name = identifier

	tok := p.nextToken()

	switch tok.Kind {
	case TokenWalrus, TokenBind:
		// fall through
	case TokenColon:
		// parse the type
		stmt.ExplicitType = p.ParseType()
		// checks for =
		if !p.expect([]TokenKind{TokenAssign}) {
			return nil, p.error(p.currentToken(), "ERROR: expected =, got shit")
		}
	default:
		return nil, p.error(tok, "ERROR: expected (:= or ::) operators, got shit")
	}

	value := p.parseExpression(LOWEST)

	if value == nil {
		return nil, p.error(tok, "ERROR: expected an expression, got nil value")
	}

	stmt.Value = value

	return stmt, nil
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
	tok := p.currentToken()

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
	prefix := p.prefixParseFns[p.currentToken().Kind]

	if prefix == nil {
		p.Errors = append(p.Errors, p.error(p.currentToken(), "ERROR: ain't an expression, it is a statement"))
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
