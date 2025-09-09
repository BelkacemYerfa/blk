package parser

import (
	"blk/ast"
	"blk/lexer"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// ? link for precedence operator https://www.tutorialspoint.com/go/go_operators_precedence.htm
const (
	_ int = iota
	LOWEST
	ASSIGN      // =
	OR          // ||
	AND         // &&
	BitOr       // |
	BitXor      // ^
	BitAnd      // &
	EQUALS      // == !=
	LESSGREATER // > < >= <=
	BitShift    // << >>
	SUM         // + -
	PRODUCT     // * / %
	PREFIX      // -X or !X or ~X
	CALL        // myFunction(X)
	INDEX       // arr[i]
	STRUCT      // Vec2{}.distance()
)

var precedences = map[lexer.TokenKind]int{
	lexer.TokenCurlyBraceOpen:      ASSIGN,
	lexer.TokenBind:                ASSIGN,
	lexer.TokenWalrus:              ASSIGN,
	lexer.TokenOr:                  OR,
	lexer.TokenAssignOr:            OR,
	lexer.TokenAnd:                 AND,
	lexer.TokenAssignAnd:           AND,
	lexer.TokenBitOr:               BitOr,
	lexer.TokenAssignBitOr:         BitOr,
	lexer.TokenBitXOR:              BitXor,
	lexer.TokenAssignBitXor:        BitXor,
	lexer.TokenBitAnd:              BitAnd,
	lexer.TokenAssignBitAnd:        BitAnd,
	lexer.TokenEquals:              EQUALS,
	lexer.TokenNotEquals:           EQUALS,
	lexer.TokenLess:                LESSGREATER,
	lexer.TokenLessOrEqual:         LESSGREATER,
	lexer.TokenGreater:             LESSGREATER,
	lexer.TokenGreaterOrEqual:      LESSGREATER,
	lexer.TokenBitRightShift:       BitShift,
	lexer.TokenAssignBitRightShift: BitShift,
	lexer.TokenBitLeftShift:        BitShift,
	lexer.TokenAssignBitLeftShift:  BitShift,
	lexer.TokenPlus:                SUM,
	lexer.TokenAssignPlus:          SUM,
	lexer.TokenAssignPlusOne:       SUM,
	lexer.TokenMinus:               SUM,
	lexer.TokenAssignMinus:         SUM,
	lexer.TokenAssignMinusOne:      SUM,
	lexer.TokenSlash:               PRODUCT,
	lexer.TokenAssignSlash:         PRODUCT,
	lexer.TokenMultiply:            PRODUCT,
	lexer.TokenAssignMultiply:      PRODUCT,
	lexer.TokenModule:              PRODUCT,
	lexer.TokenAssignModule:        PRODUCT,
	lexer.TokenExclamation:         PREFIX,
	lexer.TokenBitNot:              PREFIX,
	lexer.TokenBraceOpen:           CALL,
	lexer.TokenBracketOpen:         INDEX,
	lexer.TokenDot:                 STRUCT,
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
	p.registerPrefix(lexer.TokenSelf, p.parseIdentifier)
	p.registerPrefix(lexer.TokenInt, p.parseIntLiteral)
	p.registerPrefix(lexer.TokenFloat, p.parseFloatLiteral)
	p.registerPrefix(lexer.TokenString, p.parseStringLiteral)
	p.registerPrefix(lexer.TokenChar, p.parseCharLiteral)
	p.registerPrefix(lexer.TokenNul, p.parseNulLiteral)
	p.registerPrefix(lexer.TokenBracketOpen, p.parseArrayLiteral)
	p.registerPrefix(lexer.TokenCurlyBraceOpen, p.parseMapLiteral)
	p.registerPrefix(lexer.TokenExclamation, p.parsePrefixExpression)
	p.registerPrefix(lexer.TokenBitNot, p.parsePrefixExpression)
	p.registerPrefix(lexer.TokenMinus, p.parsePrefixExpression)
	p.registerPrefix(lexer.TokenMultiply, p.parsePrefixExpression)
	p.registerPrefix(lexer.TokenBitAnd, p.parsePrefixExpression)
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
	p.registerInfix(lexer.TokenAnd, p.parseInfixExpression)
	p.registerInfix(lexer.TokenOr, p.parseInfixExpression)
	p.registerInfix(lexer.TokenEquals, p.parseInfixExpression)
	p.registerInfix(lexer.TokenNotEquals, p.parseInfixExpression)
	p.registerInfix(lexer.TokenLess, p.parseInfixExpression)
	p.registerInfix(lexer.TokenGreater, p.parseInfixExpression)
	p.registerInfix(lexer.TokenLessOrEqual, p.parseInfixExpression)
	p.registerInfix(lexer.TokenGreaterOrEqual, p.parseInfixExpression)
	p.registerInfix(lexer.TokenBitAnd, p.parseInfixExpression)
	p.registerInfix(lexer.TokenBitOr, p.parseInfixExpression)
	p.registerInfix(lexer.TokenBitXOR, p.parseInfixExpression)
	p.registerInfix(lexer.TokenBitLeftShift, p.parseInfixExpression)
	p.registerInfix(lexer.TokenBitRightShift, p.parseInfixExpression)
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
		p.Errors = append(p.Errors, p.error(tok, fmt.Sprintf("expected one of (%v), received %v", kinds, tok.Kind)))
		return false
	}

	return true
}

func (p *Parser) error(tok lexer.Token, msg ...interface{}) error {
	errMsg := fmt.Sprintf("\033[1;90m%s:%d:%d:\033[0m ERROR: %s", p.FilePath, tok.Row, tok.Col, fmt.Sprint(msg...))

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
	case lexer.TokenNext:
		return p.parseNextStatement()
	case lexer.TokenBreak:
		return p.parseBreakStatement()
	case lexer.TokenIdentifier, lexer.TokenSelf:
		firstLook := p.lookToken(1)
		// check after it if there is a colon and a {
		if firstLook.Kind == lexer.TokenColon && p.lookToken(2).Kind == lexer.TokenCurlyBraceOpen {
			return p.parseScope()
		}

		// for the bind operations, either :: or := or : (: is for struct )
		if firstLook.Kind == lexer.TokenBind || firstLook.Kind == lexer.TokenWalrus || firstLook.Kind == lexer.TokenColon {
			return p.parseBindExpression()
		}

		// go through the current tokens in the same line until u find :: or :=, if found go to parseBindExpression, otherwise parseExpression statement

		lexerAssignOperator := []lexer.TokenKind{
			lexer.TokenAssignSlash, lexer.TokenAssignMultiply, lexer.TokenAssignModule, lexer.TokenAssignMinus, lexer.TokenAssignPlus, lexer.TokenAssignOr, lexer.TokenAssignPlus, lexer.TokenAssignBitAnd, lexer.TokenAssignBitLeftShift, lexer.TokenAssignBitRightShift, lexer.TokenAssignBitOr, lexer.TokenAssignBitXor,
		}

		lexerAssignPlusOperator := []lexer.TokenKind{
			lexer.TokenAssignPlusOne, lexer.TokenAssignMinusOne,
		}

		lexerBindOperators := []lexer.TokenKind{
			lexer.TokenAssign, lexer.TokenWalrus, lexer.TokenBind,
		}

		// merge all of them
		breakToken := slices.Concat(lexerAssignOperator, lexerAssignPlusOperator, lexerBindOperators)

		idx := 1
		for idx < len(p.Tokens) {
			token := p.lookToken(idx)

			// Break on assignment operators
			if slices.Contains(breakToken, token.Kind) {
				break
			}

			// Break on row change
			if token.Row != firstLook.Row || token.Kind == lexer.TokenEOF {
				break
			}

			idx++
		}

		if p.lookToken(idx).Kind == lexer.TokenBind || p.lookToken(idx).Kind == lexer.TokenWalrus {
			return p.parseBindExpression()
		}

		if p.lookToken(idx).Kind == lexer.TokenAssign {
			return p.parseAssignStatement()
		}

		if slices.Contains(lexerAssignOperator, p.lookToken(idx).Kind) {
			return p.parseAssignOperatorExpression()
		}

		if slices.Contains(lexerAssignPlusOperator, p.lookToken(idx).Kind) {
			return p.parseDoubleOperatorExpression()
		}

		return p.parseExpressionStatement()
	default:
		return p.parseExpressionStatement()
	}
}

func (p *Parser) parseCompositeType(prev lexer.Token) (*ast.CompositeType, error) {
	// composite type
	tp := &ast.CompositeType{Token: prev}
	if prev.Kind == lexer.TokenArray {
		tp.Kind = ast.TypeArray
	} else {
		tp.Kind = ast.TypeMap
	}

	tok := p.nextToken()
	if tok.Kind != lexer.TokenBraceOpen {
		return nil, p.error(tok, "expected ( after ", tok.Kind, " instead got ", tok.Text)
	}

	// consume the ( token
	tok = p.currentToken()
	if _, ok := lexer.TypeKeywords[tok.Kind]; !ok {
		return nil, p.error(tok, "expected a type, but instead got ", tok.Text)
	}

	// call the parse type on the current token
	nestedType, err := p.parseType()
	if err != nil {
		return nil, err
	}
	tp.LeftType = nestedType

	//switch based on the current type
	if tp.Kind == ast.TypeArray {
		// parse the array size if it exists
		tok = p.currentToken()
		// check first if next token is ) or not
		if tok.Kind != lexer.TokenBraceClose {
			// expect first a comma

			if tok.Kind != lexer.TokenComma {
				return nil, p.error(tok, "expected a comma after the ", tp.LeftType, " instead got ", tok.Text)
			}

			p.nextToken()
			tok = p.currentToken()
			// it can be an identifier also
			if tok.Kind != lexer.TokenIdentifier && tok.Kind != lexer.TokenInt {
				return nil, p.error(tok, "expected an int literal | identifier after ", tp.LeftType, " instead got ", tok.Text)
			}

			// parse that token
			expr := p.parseExpression(LOWEST)
			tp.Size = expr

			// consume the close bracket
			if p.currentToken().Kind != lexer.TokenBraceClose {
				return nil, p.error(tok, "after size in", tp, " a ) is expected, instead got ", tok.Text)
			}

		} else if tok.Kind == lexer.TokenBraceClose {
			// set the size to be -1, means dynamic
			tp.Size = &ast.IntegerLiteral{
				Token: tok,
				Value: -1,
			}
		}

	} else {
		// parse the second map type
		tok = p.nextToken()
		// first we expect the comma token
		if tok.Kind != lexer.TokenComma {
			return nil, p.error(tok, "expected a comma after the ", tp.LeftType, " instead got ", tok.Text)
		}

		// parse the second type
		nestedType, err := p.parseType()
		if err != nil {
			return nil, err
		}
		tp.RightType = nestedType

		tok = p.currentToken()
		// check the end token is )
		if tok.Kind != lexer.TokenBraceClose {
			return nil, p.error(tok, "after value type ", tp.RightType, " a ) is expected, instead got ", tok.Text)
		}
	}

	// consume the ) token
	p.nextToken()

	return tp, nil
}

func (p *Parser) parseFunctionType(prev lexer.Token) (ast.Type, error) {
	tp := &ast.FunctionType{Token: prev, Kind: ast.TypeFunction}

	tok := p.nextToken()
	if tok.Kind != lexer.TokenBraceOpen {
		return nil, p.error(tok, "expected ( after ", tp.Token.Kind, " instead got ", tok.Text)
	}

	args := make([]ast.Type, 0)

	// check if ) is after (, means no arguments
	if p.currentToken().Kind == lexer.TokenBraceClose {
		tp.Args = args
	} else {
		args, err := p.parseParamType()

		if err != nil {
			return nil, err
		}

		tp.Args = args
	}

	// consume ) token
	p.nextToken()
	tok = p.currentToken()

	if tok.Kind != lexer.TokenColon {
		return nil, p.error(tok, "expected : after ) instead got ", tok.Text)
	}

	// consume the : token
	p.nextToken()

	tok = p.currentToken()
	if tok.Kind != lexer.TokenBraceOpen {
		return nil, p.error(tok, "expected ( after ", tp.Token.Kind, " instead got ", tok.Text)
	}

	// consume the ( token
	p.nextToken()

	rets := make([]ast.Type, 0)

	if p.currentToken().Kind == lexer.TokenBraceClose {
		tp.Return = rets
	} else {
		rets, err := p.parseParamType()

		if err != nil {
			return nil, err
		}

		tp.Return = rets
	}

	// consume ) token
	p.nextToken()

	return tp, nil
}

func (p *Parser) parseParamType() ([]ast.Type, error) {
	pmType := make([]ast.Type, 0)

	fRet, err := p.parseType()
	if err != nil {
		return nil, err
	}
	pmType = append(pmType, fRet)

	for p.currentToken().Kind == lexer.TokenComma {
		p.nextToken()
		fRet, err := p.parseType()
		if err != nil {
			return nil, err
		}
		pmType = append(pmType, fRet)
	}

	// check close brace
	tok := p.currentToken()

	if tok.Kind != lexer.TokenBraceClose {
		return nil, p.error(tok, "expected ) after last argument type instead got ", tok.Text)
	}

	return pmType, nil
}

func (p *Parser) parsePointerType(tok lexer.Token) (ast.Type, error) {
	ptr := &ast.PointerType{Token: tok, Kind: ast.TypePointer}

	rht, err := p.parseType()
	if err != nil {
		return nil, err
	}

	ptr.Right = rht

	return ptr, nil
}

func (p *Parser) parsePrimitiveType(tok lexer.Token) (ast.Type, error) {
	primitive := &ast.PrimitiveType{
		Token: tok,
	}
	// parse primitive type

	switch tok.Kind {
	case lexer.TokenBool, lexer.TokenString, lexer.TokenChar:
		primitive.Size = -1

	case lexer.TokenInt8, lexer.TokenInt16, lexer.TokenInt32, lexer.TokenInt64:
		// signed int
		size, _ := strconv.ParseInt(strings.Split(tok.Text, "i")[1], 10, 8)
		primitive.Size = int(size)
		primitive.Signed = true

	case lexer.TokenUInt8, lexer.TokenUInt16, lexer.TokenUInt32, lexer.TokenUInt64, lexer.TokenFloat32, lexer.TokenFloat64:
		// unsigned int & floats
		separator := "u"
		if tok.Kind == lexer.TokenFloat64 || tok.Kind == lexer.TokenFloat32 {
			separator = "f"
		}
		size, _ := strconv.ParseInt(strings.Split(tok.Text, separator)[1], 10, 8)
		primitive.Size = int(size)

	default:
		// unsupported way for type
		if tok.Kind == lexer.TokenEnum {
			return nil, p.error(tok, "enums must be named")
		}
		// unrecognized token
		return nil, p.error(tok, "unrecognized type ", tok.Text)
	}

	return primitive, nil
}

// parses explicit types
func (p *Parser) parseType() (ast.Type, error) {
	// current token
	tok := p.nextToken()

	switch tok.Kind {
	case lexer.TokenArray, lexer.TokenMap:
		// composite type
		return p.parseCompositeType(tok)

	case lexer.TokenFn:
		// function type
		return p.parseFunctionType(tok)

	case lexer.TokenMultiply:
		// pointer type
		return p.parsePointerType(tok)

	case lexer.TokenStruct:
		// anonymous struct
		p.Pos--
		exp := p.parseStructExpression()

		if exp == nil {
			return nil, nil
		}
		return exp.(*ast.StructExpression), nil

	case lexer.TokenIdentifier:
		// TODO: think about identifier for type aliases

	default:
		// primitive type
		return p.parsePrimitiveType(tok)
	}

	return nil, nil
}

func (p *Parser) parseVarDeclaration() (*ast.VarDeclaration, error) {
	stmt := &ast.VarDeclaration{Token: p.currentToken()}
	stmt.Mutable = stmt.Token.Kind == lexer.TokenLet

	p.nextToken()

	stmt.Name = p.parseIdentifiers()

	tok := p.nextToken()

	if tok.Kind != lexer.TokenColon {
		return nil, p.error(tok, "expected assign (=), got ", tok.Text)
	}

	// type
	tp, err := p.parseType()
	if err != nil {
		return nil, err
	}

	stmt.Type = tp

	tok = p.nextToken()
	if tok.Kind != lexer.TokenAssign {
		return nil, p.error(tok, "expected assign (=), got ", tok.Text)
	}

	// TODO: change this later to support multi value
	stmt.Value = p.parseExpression(LOWEST)
	return stmt, nil
}

func (p *Parser) parseReturnStatement() (*ast.ReturnStatement, error) {
	stmt := &ast.ReturnStatement{Token: p.currentToken()}
	p.nextToken()
	returnValues := make([]ast.Expression, 0)

	if p.currentToken().Kind == lexer.TokenCurlyBraceClose {
		stmt.ReturnValues = returnValues
		return stmt, nil
	}

	// TODO: problem here with ++ & -- parsing also for <op>= exp
	// ? Suggested fix: register the ++ && -- & other precedence operation to deal with this, resulting in change in the ast.AssignmentStatement to ast.AssignmentExpression

	returnValues = append(returnValues, p.parseExpression(LOWEST))

	for p.currentToken().Kind == lexer.TokenComma {
		// consume the , token
		p.nextToken()
		returnValues = append(returnValues, p.parseExpression(LOWEST))
	}

	stmt.ReturnValues = returnValues
	return stmt, nil
}

func (p *Parser) parseImportStatement() (*ast.ImportStatement, error) {
	stmt := &ast.ImportStatement{Token: p.currentToken()}
	// skip import
	p.nextToken()

	// get the current after tok
	tok := p.currentToken()

	if tok.Kind != lexer.TokenString {
		return nil, p.error(tok, "expected a string as module name, got shit")
	}

	stmt.ModuleName = p.parseStringLiteral().(*ast.StringLiteral)

	if p.currentToken().Kind == lexer.TokenAs {
		// alias for the namespace
		p.nextToken()
		// bind the alias to it
		ident := p.parseIdentifier()
		if ident == nil {
			return nil, p.Errors[len(p.Errors)-1]
		}

		stmt.Alias = ident.(*ast.Identifier)
	}

	return stmt, nil
}

func (p *Parser) parseExpressionStatement() (*ast.ExpressionStatement, error) {
	stmt := &ast.ExpressionStatement{Token: p.currentToken()}

	expr := p.parseExpression(LOWEST)

	if expr == nil {
		return nil, fmt.Errorf("on the ast.Expression stmt")
	}

	stmt.Expression = expr

	return stmt, nil
}

func (p *Parser) parseStructExpression() ast.Expression {
	expr := &ast.StructExpression{Token: p.currentToken()}
	p.nextToken()

	if !p.expect([]lexer.TokenKind{lexer.TokenCurlyBraceOpen}) {
		p.Errors = append(p.Errors, p.error(p.currentToken(), "expected curl, got shit"))
		return nil
	}

	tok := p.currentToken()

	if tok.Kind == lexer.TokenBracketClose {
		p.nextToken()
		return &ast.StructExpression{
			Token:   expr.Token,
			Fields:  []*ast.VarDeclaration{},
			Methods: []*ast.Method{},
		}
	}

	expr.Fields, expr.Methods = p.parseFields()

	return expr
}

func (p *Parser) parseFields() ([]*ast.VarDeclaration, []*ast.Method) {
	fields := make([]*ast.VarDeclaration, 0)
	methods := make([]*ast.Method, 0)

	operatorToken := p.lookToken(1)

	switch operatorToken.Kind {
	case lexer.TokenBind:
		// parse it as method
		// means it is a function
		method, ok := p.parseIdentifier().(*ast.Identifier)
		if !ok {
			p.Errors = append(p.Errors, p.error(p.lookToken(-1), "expected an identifier, got shit"))
			return nil, nil
		}

		// this to consume the :: token
		p.nextToken()

		methods = append(methods, &ast.Method{
			Key:   method,
			Value: p.parseFunctionExpression().(*ast.FunctionExpression),
		})

	case lexer.TokenColon:
		// parse type
		field := &ast.VarDeclaration{Token: operatorToken, Mutable: true}

		ident, ok := p.parseIdentifier().(*ast.Identifier)

		if !ok {
			return nil, nil
		}

		field.Name = append(field.Name, ident)

		// consume the :
		p.nextToken()

		tp, err := p.parseType()

		if err != nil {
			p.Errors = append(p.Errors, err)
			return nil, nil
		}

		field.Type = tp

		// check if there is a default value or not
		tok := p.currentToken()

		if tok.Kind == lexer.TokenAssign {
			p.nextToken() // consume = token

			val := p.parseExpression(LOWEST)

			if val == nil {
				return nil, nil
			}
			field.Value = val
		}

		fields = append(fields, field)

	case lexer.TokenWalrus:
		// parse it as var declaration
		field, err := p.parseBindExpression()
		if err != nil {
			p.Errors = append(p.Errors, err)
			return nil, nil
		}
		fields = append(fields, field.(*ast.VarDeclaration))
	default:
		// throw an error here
		errMsg := fmt.Sprintf("expected either := or : got %s", operatorToken.Kind)
		p.Errors = append(p.Errors, p.error(operatorToken, errMsg))
		return nil, nil
	}

	for p.currentToken().Kind == lexer.TokenComma {
		p.nextToken()
		operatorToken := p.lookToken(1)

		switch operatorToken.Kind {
		case lexer.TokenBind:
			// parse it as method
			method, ok := p.parseIdentifier().(*ast.Identifier)
			if !ok {
				p.Errors = append(p.Errors, p.error(p.lookToken(-1), "expected an identifier, got shit"))
				return nil, nil
			}

			// this to consume the :: token
			p.nextToken()

			methods = append(methods, &ast.Method{
				Key:   method,
				Value: p.parseFunctionExpression().(*ast.FunctionExpression),
			})

		case lexer.TokenColon:
			// parse type
			field := &ast.VarDeclaration{Token: operatorToken, Mutable: true}

			ident, ok := p.parseIdentifier().(*ast.Identifier)

			if !ok {
				return nil, nil
			}

			field.Name = append(field.Name, ident)

			// consume the :
			p.nextToken()

			tp, err := p.parseType()

			if err != nil {
				p.Errors = append(p.Errors, err)
				return nil, nil
			}

			field.Type = tp

			// check if there is a default value or not
			tok := p.currentToken()

			if tok.Kind == lexer.TokenAssign {
				p.nextToken() // consume = token

				val := p.parseExpression(LOWEST)

				if val == nil {
					return nil, nil
				}
				field.Value = val
			}

			fields = append(fields, field)

		case lexer.TokenWalrus:
			// parse it as var declaration
			field, err := p.parseBindExpression()
			if err != nil {
				p.Errors = append(p.Errors, err)
				return nil, nil
			}
			fields = append(fields, field.(*ast.VarDeclaration))
		default:
			// throw an error here
			errMsg := fmt.Sprintf("expected either := or : got %s", operatorToken.Kind)
			p.Errors = append(p.Errors, p.error(operatorToken, errMsg))
			return nil, nil
		}
	}

	if !p.expect([]lexer.TokenKind{lexer.TokenCurlyBraceClose}) {
		p.Errors = append(p.Errors, p.error(p.currentToken(), "expected close curly brace ( } ), got shit"))
		return nil, nil
	}

	return fields, methods
}

func (p *Parser) parseEnumExpression() ast.Expression {
	expr := &ast.EnumExpression{Token: p.currentToken()}

	// consume the enum lexer.token
	p.nextToken()

	if !p.expect([]lexer.TokenKind{lexer.TokenCurlyBraceOpen}) {
		p.Errors = append(p.Errors, p.error(p.currentToken(), "expected curl, got shit"))
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
		p.Errors = append(p.Errors, p.error(p.lookToken(-1), "expected an identifier, got shit"))
		return nil
	}

	fields = append(fields, field)

	for p.currentToken().Kind == lexer.TokenComma {
		p.nextToken()
		field, ok := p.parseIdentifier().(*ast.Identifier)

		if !ok {
			p.Errors = append(p.Errors, p.error(p.lookToken(-1), "expected an identifier, got shit"))
			return nil
		}

		fields = append(fields, field)
	}

	if !p.expect([]lexer.TokenKind{lexer.TokenCurlyBraceClose}) {
		p.Errors = append(p.Errors, p.error(p.currentToken(), "expected close curly brace ( } ), got shit"))
		return nil
	}

	return fields
}

func (p *Parser) parseWhileStatement() (*ast.WhileStatement, error) {
	stmt := &ast.WhileStatement{Token: p.currentToken()}
	p.nextToken()

	stmt.Condition = p.parseExpression(ASSIGN)

	if !p.expect([]lexer.TokenKind{lexer.TokenCurlyBraceOpen}) {
		return nil, fmt.Errorf("expected curly brace open ( { ), got shit")
	}

	stmt.Body = p.parseBlockStatement().(*ast.BlockStatement)
	return stmt, nil
}

func (p *Parser) parseForStatement() (*ast.ForStatement, error) {
	stmt := &ast.ForStatement{Token: p.currentToken()}
	p.nextToken()

	tok := p.currentToken()

	if tok.Kind != lexer.TokenIdentifier {
		return nil, p.error(tok, "expected at least one identifier, got shit")
	}

	stmt.Identifiers = append(stmt.Identifiers, p.parseIdentifier().(*ast.Identifier))

	tok = p.nextToken()

	if tok.Kind == lexer.TokenComma {
		ident, ok := p.parseIdentifier().(*ast.Identifier)
		if !ok {
			return nil, p.error(tok, "expected an identifier, got shit")
		}
		stmt.Identifiers = append(stmt.Identifiers, ident)
	} else {
		p.Pos--
	}

	tok = p.nextToken()
	if tok.Kind != lexer.TokenIn {
		return nil, p.error(tok, "expected in, got shit")
	}

	// look ahead and see if the pattern <number>..<number>
	if p.lookToken(1).Kind == lexer.TokenRange {
		// use the range pattern struct fro the ast (ast.RangePattern)
		pattern := &ast.RangePattern{Token: p.currentToken()}
		pattern.Start = p.parseExpression(OR)
		tok := p.nextToken()
		if tok.Kind != lexer.TokenRange {
			return nil, p.error(tok, "expected .. token, got shit")
		}
		// if operator exists it's only assign (=)
		operatorToken := p.lookToken(0)
		if operatorToken.Kind == lexer.TokenAssign {
			pattern.Op = operatorToken.Text
			p.nextToken() // consume the operator
		} else {
			if _, ok := lexer.BinOperators[operatorToken.Kind]; ok {
				return nil, p.error(tok, "only allowed operator is =, got shit")
			}
		}

		pattern.End = p.parseExpression(OR)
		stmt.Target = pattern
	} else {
		stmt.Target = p.parseExpression(OR)
	}

	if !p.expect([]lexer.TokenKind{lexer.TokenCurlyBraceOpen}) {
		return nil, p.error(p.currentToken(), "expected curly brace open ( { ), got shit")
	}

	stmt.Body = p.parseBlockStatement().(*ast.BlockStatement)
	return stmt, nil
}

func (p *Parser) parseNextStatement() (*ast.NextStatement, error) {
	stmt := &ast.NextStatement{Token: p.currentToken()}
	// consume the next token
	p.nextToken()
	return stmt, nil
}

func (p *Parser) parseBreakStatement() (*ast.BreakStatement, error) {
	stmt := &ast.BreakStatement{Token: p.currentToken()}
	// consume the break token
	p.nextToken()
	return stmt, nil
}

func (p *Parser) parseAssignStatement() (*ast.AssignStatement, error) {
	stmt := &ast.AssignStatement{Token: p.currentToken()}

	stmt.Left = p.parsePrefixExpressionWrapper()

	// check for the token assign
	tok := p.nextToken()

	if tok.Kind != lexer.TokenAssign {
		return nil, p.error(tok, "expected assign token (=), got shit")
	}

	stmt.Right = p.parsePrefixExpressionWrapper()

	return stmt, nil
}

func (p *Parser) parsePrefixExpressionWrapper() []ast.Expression {
	exps := make([]ast.Expression, 0)

	exps = append(exps, p.parseExpression(LOWEST))

	for p.currentToken().Kind == lexer.TokenComma {
		// consume the comma (,) token
		p.nextToken()
		exps = append(exps, p.parseExpression(LOWEST))
	}

	return exps
}

func (p *Parser) parseIdentifier() ast.Expression {
	tok := p.nextToken()

	if tok.Kind != lexer.TokenIdentifier && tok.Kind != lexer.TokenSelf {
		p.Errors = append(p.Errors, p.error(tok, "expected identifier, got shit"))
		return nil
	}

	return &ast.Identifier{
		Token: tok,
		Value: tok.Text,
	}
}

// this function parses multi identifiers
// attempt to gradually shift into supporting multi values
func (p *Parser) parseIdentifiers() []*ast.Identifier {
	identifiers := make([]*ast.Identifier, 0)

	ident := p.parseIdentifier()
	if ident == nil {
		return identifiers
	}

	identifiers = append(identifiers, ident.(*ast.Identifier))

	for p.currentToken().Kind == lexer.TokenComma {
		p.nextToken()

		ident := p.parseIdentifier()
		if ident == nil {
			return identifiers
		}

		identifiers = append(identifiers, ident.(*ast.Identifier))
	}

	return identifiers
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

func (p *Parser) parseCharLiteral() ast.Expression {
	tok := p.nextToken()
	code, _, _, err := strconv.UnquoteChar(tok.Text, '\'')
	if err != nil {
		p.Errors = append(p.Errors, p.error(tok, err.Error()))
		return nil
	}
	return &ast.CharLiteral{
		Token: tok,
		Value: code,
	}
}

func (p *Parser) parseNulLiteral() ast.Expression {
	tok := p.nextToken()
	return &ast.NulLiteral{
		Token: tok,
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
	expr := &ast.ArrayLiteral{Token: p.currentToken()}

	if !p.expect([]lexer.TokenKind{lexer.TokenBracketOpen}) {
		p.Errors = append(p.Errors, p.error(expr.Token, "expected open bracket [, got shit"))
		return nil
	}

	elements := make([]ast.Expression, 0)

	// check for size of the array
	if p.lookToken(1).Text == ";" {
		// means the first part is the size
		expr.Size = p.parseExpression(LOWEST)
		// consume the ; token
		p.nextToken()
	}

	tok := p.currentToken()

	if tok.Kind == lexer.TokenBracketClose {
		p.nextToken()
		return &ast.ArrayLiteral{
			Token:    expr.Token,
			Elements: elements,
		}
	}

	elements = append(elements, p.parseExpression(LOWEST))

	for p.currentToken().Kind == lexer.TokenComma {
		p.nextToken()
		elements = append(elements, p.parseExpression(LOWEST))
	}

	expr.Elements = elements

	if !p.expect([]lexer.TokenKind{lexer.TokenBracketClose}) {
		p.Errors = append(p.Errors, p.error(p.currentToken(), "expected close bracket ( ] ), got shit"))
		return nil
	}

	return expr
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
		return nil, p.error(tok, "expected colon (:), got shit")
	}

	if !p.expect([]lexer.TokenKind{lexer.TokenCurlyBraceOpen}) {
		tok := p.currentToken()
		return nil, p.error(tok, "expected ({), got shit")
	}

	stmt.Body = p.parseBlockStatement().(*ast.BlockStatement)
	return stmt, nil
}

func (p *Parser) parseMapLiteral() ast.Expression {
	prev := p.currentToken()

	if !p.expect([]lexer.TokenKind{lexer.TokenCurlyBraceOpen}) {
		p.Errors = append(p.Errors, p.error(prev, "expected open curly- brace {, got shit"))
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
		p.Errors = append(p.Errors, p.error(tok, "expected colon ( : ), got shit"))
		return nil
	}

	pairs[key] = p.parseExpression(LOWEST)

	for p.currentToken().Kind == lexer.TokenComma {
		p.nextToken()
		key := p.parseExpression(LOWEST)

		tok = p.nextToken()

		if tok.Kind != lexer.TokenColon {
			p.Errors = append(p.Errors, p.error(tok, "expected colon ( : ), got shit"))
			return nil
		}

		pairs[key] = p.parseExpression(LOWEST)
	}

	if !p.expect([]lexer.TokenKind{lexer.TokenCurlyBraceClose}) {
		p.Errors = append(p.Errors, p.error(prev, "expected close curly brace ( } ), got shit"))
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

	// look ahead to the next token
	if p.currentToken().Kind == lexer.TokenQuestion || p.currentToken().Kind == lexer.TokenUse {
		// do something
		p.nextToken() // consume the ?
		exprStmt, err := p.parseExpressionStatement()
		if err != nil {
			return nil
		}
		expr.Consequence = &ast.BlockStatement{
			Body: []ast.Statement{exprStmt},
		}
		tok := p.nextToken()
		if tok.Kind != lexer.TokenColon && tok.Kind != lexer.TokenElse {
			p.Errors = append(p.Errors, p.error(p.currentToken(), "expected else or : as following token for the ternary definition, got ", tok.Kind))
			return nil
		}
		// fill the alternative case
		exprStmt, err = p.parseExpressionStatement()
		if err != nil {
			return nil
		}
		expr.Alternative = &ast.BlockStatement{
			Body: []ast.Statement{exprStmt},
		}
	} else {
		if !p.expect([]lexer.TokenKind{lexer.TokenCurlyBraceOpen}) {
			p.Errors = append(p.Errors, p.error(p.currentToken(), "expected close curly brace ( } ), got shit"))
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
	}

	return expr
}

func (p *Parser) parseMatchExpression() ast.Expression {
	expr := &ast.MatchExpression{Token: p.currentToken()}
	// consume math keyword
	p.nextToken()

	expr.MatchKey = p.parseExpression(ASSIGN)

	if !p.expect([]lexer.TokenKind{lexer.TokenCurlyBraceOpen}) {
		p.Errors = append(p.Errors, p.error(p.currentToken(), "expected close curly brace '{', got shit"))
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
		p.Errors = append(p.Errors, p.error(tok, "expected colon (match), got shit"))
		return nil
	}

	if !p.expect([]lexer.TokenKind{lexer.TokenCurlyBraceOpen}) {
		p.Errors = append(p.Errors, p.error(p.currentToken(), "expected close curly brace '{', got shit"))
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
			p.Errors = append(p.Errors, p.error(tok, "expected colon ( : ), got shit"))
			return nil
		}

		if !p.expect([]lexer.TokenKind{lexer.TokenCurlyBraceOpen}) {
			p.Errors = append(p.Errors, p.error(p.currentToken(), "expected close curly brace '{', got shit"))
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
		p.Errors = append(p.Errors, p.error(p.currentToken(), "expected close curly brace '}', got shit"))
		return nil
	}

	return expr
}

func (p *Parser) parseFunctionExpression() ast.Expression {
	expr := &ast.FunctionExpression{Token: p.currentToken()}
	p.nextToken()

	if !p.expect([]lexer.TokenKind{lexer.TokenBraceOpen}) {
		p.Errors = append(p.Errors, p.error(p.currentToken(), "expected brace open '(' , got shit"))
		return nil
	}

	self, args := p.parseArguments()

	if args == nil {
		p.Errors = append(p.Errors, p.error(p.currentToken(), "expected arguments, got shit"))
		return nil
	}

	// isn't required to exist
	expr.Self = self
	expr.Args = args

	if !p.expect([]lexer.TokenKind{lexer.TokenCurlyBraceOpen}) {
		p.Errors = append(p.Errors, fmt.Errorf("expected curly brace open ( { ), got shit"))
		return nil
	}

	body := p.parseBlockStatement().(*ast.BlockStatement)

	if body == nil {
		p.Errors = append(p.Errors, p.error(p.currentToken(), "expected valid body, got shit"))
		return nil
	}

	expr.Body = body

	return expr
}

func (p *Parser) parseArguments() (*ast.Identifier, []*ast.Identifier) {
	// return another identifier which is
	args := make([]*ast.Identifier, 0)
	self := &ast.Identifier{}

	// self needs to be defined at first
	if p.currentToken().Kind == lexer.TokenSelf {
		self.Token = p.currentToken()
		self.Value = p.currentToken().Text
		// consume the self
		p.nextToken()
		if p.currentToken().Kind == lexer.TokenComma {
			// consume the comma
			p.nextToken()
		}
	}

	if p.currentToken().Kind == lexer.TokenBraceClose {
		p.nextToken()
		return self, args
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
		return nil, nil
	}

	return self, args
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
		p.Errors = append(p.Errors, p.error(p.currentToken(), "only call are allowed, bounding function into a variable ain't allowed"))
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
		expr := p.parseExpression(LOWEST)
		if expr == nil {
			return nil
		}
		args = append(args, expr)
	}

	if !p.expect([]lexer.TokenKind{lexer.TokenBraceClose}) {
		return nil
	}

	return args
}

func (p *Parser) parseIndexExpression(left ast.Expression) ast.Expression {
	exp := &ast.IndexExpression{Token: left.GetToken(), Left: left}
	p.nextToken()

	if p.currentToken().Kind != lexer.TokenColon {
		exp.Start = p.parseExpression(LOWEST)
	}

	if p.currentToken().Kind == lexer.TokenColon {
		exp.Range = true
		p.nextToken() // consume :
	}

	if p.currentToken().Kind != lexer.TokenBracketClose {
		exp.End = p.parseExpression(LOWEST)
	}

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

	tok := p.currentToken()
	if tok.Kind == lexer.TokenCurlyBraceClose {
		p.nextToken()
		return fields
	}

	identifier, ok := p.parseIdentifier().(*ast.Identifier)

	if !ok {
		p.Errors = append(p.Errors, p.error(p.currentToken(), "expected an identifier, got shit"))
		return []ast.FieldInstance{}
	}

	tok = p.nextToken()

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

	// the precedence needs to be >= () function call
	expr.Property = p.parseExpression(PREFIX)
	// ? if these results in more bugs consider changing it to a binary expression where the operator is a .
	// then in the evaluation layer we see what operator is it, and then do something

	return expr
}

// this function is responsible to parsing the assign operator syntax
// an example of this: index += 1 <=> index = index + 1
func (p *Parser) parseAssignOperatorExpression() (*ast.AssignStatement, error) {
	expr := &ast.AssignStatement{Token: p.currentToken()}

	expr.Left = p.parsePrefixExpressionWrapper()
	// get the operator, from the current op which can be something (+=,%=,..etc)
	operator := strings.Split(p.currentToken().Text, "=")[0]
	// consume the operator token
	p.nextToken()
	// parse the operator
	expr.Right = []ast.Expression{
		&ast.BinaryExpression{
			Token:    p.currentToken(),
			Operator: operator,
			Left:     expr.Left[0],
			Right:    p.parseExpression(LOWEST),
		},
	}

	return expr, nil
}

// this function is responsible of parsing the double operator assign
// an example of this : index++, index-- <=> index = index + 1
// only support for (+,-) operators
func (p *Parser) parseDoubleOperatorExpression() (*ast.AssignStatement, error) {
	expr := &ast.AssignStatement{Token: p.currentToken()}

	expr.Left = p.parsePrefixExpressionWrapper()
	// get the operator, from the current op which can be something (+=,%=,..etc)
	operator := string(p.currentToken().Text[0])

	// parse the operator
	expr.Right = []ast.Expression{
		&ast.BinaryExpression{
			Token:    p.currentToken(),
			Operator: operator,
			Left:     expr.Left[0],
			// default of it this
			Right: &ast.IntegerLiteral{
				Value: 1,
			},
		},
	}

	// consume the operator token (++, --)
	p.nextToken()

	return expr, nil
}

func (p *Parser) parseBindExpression() (ast.Statement, error) {
	stmt := &ast.VarDeclaration{Token: lexer.Token{
		LiteralToken: lexer.LiteralToken{
			Text: "let",
			Kind: lexer.TokenLet,
		},
		Col: p.currentToken().Col,
		Row: p.currentToken().Row,
	}, Mutable: true}

	stmt.Name = p.parseIdentifiers()

	tok := p.nextToken()

	switch tok.Kind {
	case lexer.TokenBind:
		stmt.Token = lexer.Token{
			LiteralToken: lexer.LiteralToken{
				Text: "const",
				Kind: lexer.TokenConst,
			},
		}
		stmt.Mutable = false
		// fall through
	case lexer.TokenWalrus:
		// fall through
	case lexer.TokenColon:
		// consume the :

		tp, err := p.parseType()
		if err != nil {
			return nil, err
		}

		stmt.Type = tp

		tok = p.nextToken()

		if tok.Kind != lexer.TokenAssign && tok.Kind != lexer.TokenColon {
			return nil, p.error(tok, "expected assign (= | :) after", tp, " token got ", tok.Text)
		}

		if tok.Kind == lexer.TokenColon {
			stmt.Token = lexer.Token{
				LiteralToken: lexer.LiteralToken{
					Text: "const",
					Kind: lexer.TokenConst,
				},
			}
			stmt.Mutable = false
		}

	default:
		return nil, p.error(tok, "expected (:= or ::) operators, got shit")
	}

	value := p.parseExpression(LOWEST)

	if value == nil {
		return nil, p.error(tok, "expected an ast.Expression, got nil value")
	}

	stmt.Value = value

	return stmt, nil
}

func (p *Parser) parsePrefixExpression() ast.Expression {
	tok := p.nextToken()

	if _, ok := lexer.UnaryOperators[tok.Kind]; !ok {
		p.Errors = append(p.Errors, p.error(tok, "expected a unary operator (! | -), got shut"))
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
		p.Errors = append(p.Errors, p.error(tok, "expected a binary operator (== | > | < | ...), got shut"))
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
	cur := p.currentToken()
	if cur.Kind == lexer.TokenError {
		p.Errors = append(p.Errors, p.error(cur, cur.Text))
		return nil
	}

	prefix := p.prefixParseFns[cur.Kind]

	if prefix == nil {
		p.Errors = append(p.Errors, p.error(cur, "ain't an ast.Expression, it is a statement"))
		return nil
	}

	leftExp := prefix()
	cur = p.currentToken()

	if cur.Kind == lexer.TokenBraceOpen {
		// make sure that the token before is an identifier
		lookBeforeKind := p.lookToken(-1).Kind
		_, ok := lexer.BinOperators[lookBeforeKind]
		if lookBeforeKind != lexer.TokenIdentifier && !ok && cur.Col > 1 {
			p.Errors = append(p.Errors, p.error(p.currentToken(), "brace token expects to be an identifier before it, or a binary operator"))
			return nil
		}
	}

	for p.currentToken().Row <= cur.Row && p.currentToken().Kind != lexer.TokenEOF && precedence < p.peekPrecedence() && p.lookToken(-1).Row == cur.Row {
		infix := p.infixParseFns[p.currentToken().Kind]
		if infix == nil {
			return leftExp
		}
		leftExp = infix(leftExp)
	}

	return leftExp
}
