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
	lexer.TokenAssign:              ASSIGN,
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
	lexer          *lexer.Lexer
	FilePath       string
	Errors         []error
	Pos            int
	prefixParseFns map[lexer.TokenKind]prefixParseFn
	infixParseFns  map[lexer.TokenKind]infixParseFn
	internalFlags  []string

	prevToken lexer.Token // previous token of current token
	curToken  lexer.Token
	peekToken lexer.Token // one token lookahead
}

func NewParser(lex *lexer.Lexer, filepath string) *Parser {
	p := Parser{
		lexer:          lex,
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
	// TODO: add ++ & -- tokens to be sort of expression
	p.registerInfix(lexer.TokenAssignPlusOne, p.parseDoubleOperatorExpression)
	p.registerInfix(lexer.TokenAssignMinusOne, p.parseDoubleOperatorExpression)
	// TODO : add +=, -=, *=, /=, %=, &=, |=, ^=, <<=, >>=, &&=, ||=
	p.registerInfix(lexer.TokenAssignSlash, p.parseAssignOperatorExpression)
	p.registerInfix(lexer.TokenAssignMultiply, p.parseAssignOperatorExpression)
	p.registerInfix(lexer.TokenAssignModule, p.parseAssignOperatorExpression)
	p.registerInfix(lexer.TokenAssignMinus, p.parseAssignOperatorExpression)
	p.registerInfix(lexer.TokenAssignPlus, p.parseAssignOperatorExpression)
	p.registerInfix(lexer.TokenAssignOr, p.parseAssignOperatorExpression)
	p.registerInfix(lexer.TokenAssignBitOr, p.parseAssignOperatorExpression)
	p.registerInfix(lexer.TokenAssignAnd, p.parseAssignOperatorExpression)
	p.registerInfix(lexer.TokenAssignBitAnd, p.parseAssignOperatorExpression)
	p.registerInfix(lexer.TokenAssignBitLeftShift, p.parseAssignOperatorExpression)
	p.registerInfix(lexer.TokenAssignBitRightShift, p.parseAssignOperatorExpression)
	p.registerInfix(lexer.TokenAssignBitXor, p.parseAssignOperatorExpression)

	// set the tok position
	p.nextToken()
	p.nextToken()

	return &p
}

func (p *Parser) peekPrecedence() int {
	if p, ok := precedences[p.curToken.Kind]; ok {
		return p
	}
	return LOWEST
}

func (p *Parser) nextToken() {
	p.Pos++
	p.prevToken = p.curToken
	p.curToken = p.peekToken
	p.peekToken = p.lexer.NextToken()
}

func (p *Parser) add(err error) {
	if len(err.Error()) > 0 {
		p.Errors = append(p.Errors, err)
	}
}

// sync the token position by consuming all the current tokens in the current row, stop in the next row
func (p *Parser) sync(anotherStep bool) {
	for p.curToken.Row == p.prevToken.Row {
		p.nextToken()
	}

	if anotherStep {
		p.nextToken()
	}
}

// syncs until reaching the given token kind, second param is responsible of consuming the final token kind or not
// useful with complex body parsing in custom bodies, such as enums, and structs, since those don't use the parseBlockExpression function
func (p *Parser) syncUntilTokenIs(kind lexer.TokenKind, consumeFinal bool) {
	for !p.curTokenKindIs(kind) {
		p.nextToken()
	}
	if consumeFinal {
		// consume that token
		p.nextToken()
	}
}

func mapExprToIdentifiers(exprs []ast.Expression) []*ast.Identifier {
	res := make([]*ast.Identifier, 0, len(exprs))
	for _, e := range exprs {
		if ident, ok := e.(*ast.Identifier); ok {
			res = append(res, ident)
		}
	}
	return res
}

func (p *Parser) curTokenKindIs(kind lexer.TokenKind) bool {
	return p.curToken.Kind == kind
}

func (p *Parser) peekTokenKindIs(kind lexer.TokenKind) bool {
	return p.peekToken.Kind == kind
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

	for !p.curTokenKindIs(lexer.TokenEOF) {
		stmt, err := p.parseStatement()

		if err != nil {
			p.add(err)
			p.sync(true)
		} else {
			ast.Statements = append(ast.Statements, stmt)
		}
	}

	return &ast
}

// TODO: better error handling and targeting

func (p *Parser) parseStatement() (ast.Statement, error) {

	switch p.curToken.Kind {
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
	case lexer.TokenCurlyBraceOpen:
		return p.parseScope()
	case lexer.TokenIdentifier, lexer.TokenSelf:

		// for the bind operations, either :: or := or : (: is for struct )
		if p.peekTokenKindIs(lexer.TokenBind) || p.peekTokenKindIs(lexer.TokenWalrus) || p.peekTokenKindIs(lexer.TokenColon) {
			return p.parseBindExpression()
		}

		// for call expressions
		if p.peekTokenKindIs(lexer.TokenBraceOpen) {
			return p.parseExpressionStatement()
		}

		// go through the current tokens in the same line until u find :: or :=, if found go to parseBindExpression, otherwise parseExpression statement

		idents := make([]ast.Expression, 0)

		breakOperators := slices.Concat(lexer.AssignBinOps, lexer.AssignOp)

		// parse until one of those token
		for !slices.Contains(breakOperators, p.curToken.Kind) {
			if p.curTokenKindIs(lexer.TokenComma) {
				p.nextToken() // consume token comma
			}
			if !p.curTokenKindIs(lexer.TokenIdentifier) {
				return nil, p.error(p.curToken, "with assignments operators, left side expects only identifier, instead got ", p.curToken.Text)
			}

			idents = append(idents, p.parseIdentifier())
		}

		switch {
		case slices.Contains(lexer.AssignBinOps, p.curToken.Kind):
			if len(idents) > 1 {
				return nil, p.error(p.curToken, p.curToken.Text, " operators, can't have more than one lhs expression, got ", len(idents), " expressions")
			}

			expr := p.parseAssignOperatorExpression(idents[0])

			return &ast.ExpressionStatement{
				Token:      idents[0].GetToken(),
				Expression: expr,
			}, nil

		case slices.Contains(lexer.AssignOp, p.curToken.Kind):
			if p.curTokenKindIs(lexer.TokenAssign) {
				stmt := &ast.AssignStatement{Token: p.curToken, Left: idents}
				p.nextToken()

				stmt.Right = p.parsePrefixExpressionWrapper()
				return stmt, nil
			} else {

				stmt := &ast.VarDeclaration{
					Token: lexer.Token{
						LiteralToken: lexer.LiteralToken{
							Text: "let",
							Kind: lexer.TokenLet,
						},
						Col: p.curToken.Col,
						Row: p.curToken.Row,
					},
					Mutable: p.curTokenKindIs(lexer.TokenWalrus),
					Name:    mapExprToIdentifiers(idents)}

				if p.curTokenKindIs(lexer.TokenBind) {
					stmt.Token.LiteralToken = lexer.LiteralToken{
						Text: "const",
						Kind: lexer.TokenConst,
					}
				}

				// consume cur tok
				p.nextToken()

				stmt.Value = p.parseExpression(LOWEST)

				return stmt, nil
			}

		default:
			return p.parseExpressionStatement()
		}

	case lexer.TokenError:
		return nil, p.error(p.curToken, p.curToken.Text)

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

	p.nextToken()

	if !p.curTokenKindIs(lexer.TokenBraceOpen) {
		return nil, p.error(p.curToken, "expected ( after ", p.curToken.Kind, " instead got ", p.curToken.Text)
	}

	// consume the ( token
	p.nextToken()

	if _, ok := lexer.TypeKeywords[p.curToken.Kind]; !ok {
		return nil, p.error(p.curToken, "expected a type, but instead got ", p.curToken.Text)
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
		// check first if next token is ) or not
		if !p.curTokenKindIs(lexer.TokenBraceClose) {
			// expect first a comma

			if !p.curTokenKindIs(lexer.TokenComma) {
				return nil, p.error(p.curToken, "expected a comma after the ", tp.LeftType, " instead got ", p.curToken.Text)
			}

			p.nextToken()

			// it can be an identifier also
			if !p.curTokenKindIs(lexer.TokenIdentifier) && !p.curTokenKindIs(lexer.TokenInt) {
				return nil, p.error(p.curToken, "expected an int literal | identifier after ", tp.LeftType, " instead got ", p.curToken.Text)
			}

			// parse that token
			expr := p.parseExpression(LOWEST)
			tp.Size = expr

			// consume the close bracket
			if !p.curTokenKindIs(lexer.TokenBraceClose) {
				return nil, p.error(p.curToken, "after size in", tp, " a ) is expected, instead got ", p.curToken.Text)
			}

		} else if p.curTokenKindIs(lexer.TokenBraceClose) {
			// set the size to be -1, means dynamic
			tp.Size = &ast.IntegerLiteral{
				Token: p.curToken,
				Value: -1,
			}
		}

	} else {
		// parse the second map type
		// first we expect the comma token
		if !p.curTokenKindIs(lexer.TokenComma) {
			return nil, p.error(p.curToken, "expected a comma after the ", tp.LeftType, " instead got ", p.curToken.Text)
		}

		// consume the ,
		p.nextToken()

		// parse the second type
		nestedType, err := p.parseType()
		if err != nil {
			return nil, err
		}
		tp.RightType = nestedType

		// check the end token is )
		if !p.curTokenKindIs(lexer.TokenBraceClose) {
			return nil, p.error(p.curToken, "after value type ", tp.RightType, " a ) is expected, instead got ", p.curToken.Text)
		}
	}

	// consume the ) token
	p.nextToken()

	return tp, nil
}

func (p *Parser) parseFunctionType(prev lexer.Token) (ast.Type, error) {
	tp := &ast.FunctionType{Token: prev, Kind: ast.TypeFunction}

	p.nextToken()

	if !p.curTokenKindIs(lexer.TokenBraceOpen) {
		return nil, p.error(p.curToken, "expected ( after ", tp.Token.Kind, " instead got ", p.curToken.Text)
	}

	p.nextToken()

	args := make([]ast.Type, 0)

	// check if ) is after (, means no arguments
	if p.curTokenKindIs(lexer.TokenBraceClose) {
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

	if !p.curTokenKindIs(lexer.TokenColon) {
		return nil, p.error(p.curToken, "expected : after ) instead got ", p.curToken.Text)
	}

	// consume the : token
	p.nextToken()

	if !p.curTokenKindIs(lexer.TokenBraceOpen) {
		return nil, p.error(p.curToken, "expected ( after ", tp.Token.Kind, " instead got ", p.curToken.Text)
	}

	// consume the ( token
	p.nextToken()

	rets := make([]ast.Type, 0)

	if p.curTokenKindIs(lexer.TokenBraceClose) {
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

	for p.curTokenKindIs(lexer.TokenComma) {
		p.nextToken()
		fRet, err := p.parseType()
		if err != nil {
			return nil, err
		}
		pmType = append(pmType, fRet)
	}

	// check close brace

	if !p.curTokenKindIs(lexer.TokenBraceClose) {
		return nil, p.error(p.curToken, "expected ) after last argument type instead got ", p.curToken.Text)
	}

	return pmType, nil
}

func (p *Parser) parsePointerType(tok lexer.Token) (ast.Type, error) {
	ptr := &ast.PointerType{Token: tok, Kind: ast.TypePointer}

	p.nextToken()

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

	p.nextToken()

	return primitive, nil
}

// parses explicit types
func (p *Parser) parseType() (ast.Type, error) {
	// current token

	switch p.curToken.Kind {
	case lexer.TokenArray, lexer.TokenMap:
		// composite type
		return p.parseCompositeType(p.curToken)

	case lexer.TokenFn:
		// function type
		return p.parseFunctionType(p.curToken)

	case lexer.TokenMultiply:
		// pointer type
		return p.parsePointerType(p.curToken)

	case lexer.TokenStruct:
		// anonymous struct
		exp := p.parseStructExpression()

		if exp == nil {
			return nil, nil
		}
		return exp.(*ast.StructExpression), nil

	case lexer.TokenIdentifier:
		// TODO: think about identifier for type aliases

	default:
		// primitive type
		return p.parsePrimitiveType(p.curToken)
	}

	return nil, nil
}

func (p *Parser) parseVarDeclaration() (*ast.VarDeclaration, error) {
	stmt := &ast.VarDeclaration{Token: p.curToken}
	stmt.Mutable = stmt.Token.Kind == lexer.TokenLet

	p.nextToken()

	stmt.Name = p.parseIdentifiers()

	if !p.curTokenKindIs(lexer.TokenColon) {
		return nil, p.error(p.curToken, "expected colon (:), got ", p.curToken.Text)
	}

	// consume :
	p.nextToken()

	// type
	tp, err := p.parseType()
	if err != nil {
		return nil, err
	}

	stmt.Type = tp

	if !p.curTokenKindIs(lexer.TokenAssign) {
		return nil, p.error(p.curToken, "expected assign (=), got ", p.curToken.Text)
	}
	// consume =
	p.nextToken()

	// TODO: change this later to support multi value
	stmt.Value = p.parseExpression(LOWEST)
	return stmt, nil
}

func (p *Parser) parseReturnStatement() (*ast.ReturnStatement, error) {
	stmt := &ast.ReturnStatement{Token: p.curToken}
	p.nextToken()

	returnValues := make([]ast.Expression, 0)

	// TODO: problem here with ++ & -- parsing also for <op>= exp
	// ? Suggested fix: register the ++ && -- & other precedence operation to deal with this, resulting in change in the ast.AssignmentStatement to ast.AssignmentExpression

	returnValues = append(returnValues, p.parseExpression(LOWEST))

	for p.curTokenKindIs(lexer.TokenComma) {
		// consume the , token
		p.nextToken()
		returnValues = append(returnValues, p.parseExpression(LOWEST))
	}

	stmt.ReturnValues = returnValues
	return stmt, nil
}

func (p *Parser) parseImportStatement() (*ast.ImportStatement, error) {
	stmt := &ast.ImportStatement{Token: p.curToken}

	// skip import
	p.nextToken()

	if !p.curTokenKindIs(lexer.TokenString) {
		return nil, p.error(p.curToken, "expected a string as module name, instead got ", p.curToken)
	}

	p.nextToken()

	val := p.parseStringLiteral()

	if val == nil {
		return nil, fmt.Errorf("")
	}

	stmt.ModuleName = val.(*ast.StringLiteral)

	if p.curTokenKindIs(lexer.TokenAs) {
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
	stmt := &ast.ExpressionStatement{Token: p.curToken}

	expr := p.parseExpression(LOWEST)

	if expr == nil {
		return nil, fmt.Errorf("")
	}

	stmt.Expression = expr

	return stmt, nil
}

func (p *Parser) parseStructExpression() ast.Expression {
	expr := &ast.StructExpression{Token: p.curToken}
	p.nextToken()

	if !p.curTokenKindIs(lexer.TokenCurlyBraceOpen) {
		p.add(p.error(p.curToken, "expected curly brace open {, instead got ", p.curToken.Text))
		return nil
	}

	p.nextToken()

	if p.curTokenKindIs(lexer.TokenBracketClose) {
		p.nextToken()
		return &ast.StructExpression{
			Token:   expr.Token,
			Fields:  []*ast.VarDeclaration{},
			Methods: []*ast.Method{},
		}
	}

	fields, methods, err := p.parseFields()

	if err != nil {
		p.add(err)
		return nil
	}

	expr.Fields = fields
	expr.Methods = methods

	return expr
}

func (p *Parser) parseFields() ([]*ast.VarDeclaration, []*ast.Method, error) {
	fields := make([]*ast.VarDeclaration, 0)
	methods := make([]*ast.Method, 0)

	// parse until, then consume it
	for !p.curTokenKindIs(lexer.TokenCurlyBraceClose) {

		switch p.peekToken.Kind {
		case lexer.TokenBind:
			// parse it as method
			if !p.curTokenKindIs(lexer.TokenIdentifier) {
				err := p.error(p.prevToken, "expected an identifier, got ", p.prevToken.Text)
				return nil, nil, err
			}

			method := p.parseIdentifier().(*ast.Identifier)

			// this to consume the :: token
			p.nextToken()

			methods = append(methods, &ast.Method{
				Key:   method,
				Value: p.parseFunctionExpression().(*ast.FunctionExpression),
			})

			// check if there is a comma
			if !p.curTokenKindIs(lexer.TokenComma) {
				err := p.error(p.curToken, "expected an comma (,) at the end of each field, instead got ", p.prevToken.Text)
				p.add(err)
			}

			p.nextToken()

		case lexer.TokenColon:
			// parse type
			field := &ast.VarDeclaration{Token: p.peekToken, Mutable: true}

			if !p.curTokenKindIs(lexer.TokenIdentifier) {
				err := p.error(p.prevToken, "expected an identifier, got ", p.prevToken.Text)
				return nil, nil, err
			}

			ident := p.parseIdentifier().(*ast.Identifier)

			field.Name = append(field.Name, ident)

			// consume the :
			p.nextToken()

			tp, err := p.parseType()

			if err != nil {
				return nil, nil, err
			}

			field.Type = tp

			// check if there is a default value or not
			if p.curTokenKindIs(lexer.TokenAssign) {
				p.nextToken() // consume = token

				val := p.parseExpression(LOWEST)

				if val == nil {
					return nil, nil, fmt.Errorf("")
				}

				field.Value = val
			}

			fields = append(fields, field)

			// check if there is a comma
			if !p.curTokenKindIs(lexer.TokenComma) {
				err := p.error(p.curToken, "expected an comma (,) at the end of each field, instead got ", p.prevToken.Text)
				p.add(err)
			}

			p.nextToken()

		case lexer.TokenWalrus:
			// parse it as var declaration
			field, err := p.parseBindExpression()
			if err != nil {
				return nil, nil, err
			}
			fields = append(fields, field.(*ast.VarDeclaration))

			// check if there is a comma
			if !p.curTokenKindIs(lexer.TokenComma) {
				err := p.error(p.curToken, "expected an comma (,) at the end of each field, instead got ", p.prevToken.Text)
				p.add(err)
			}

			p.nextToken()

		default:
			// throw an error here
			err := p.error(p.curToken, "expected either (:= | :: | :), instead got ", p.curToken.Text)
			p.add(err)
			p.nextToken()
		}
	}

	p.nextToken()

	return fields, methods, nil
}

func (p *Parser) parseEnumExpression() ast.Expression {
	expr := &ast.EnumExpression{Token: p.curToken}

	// consume the enum lexer.token
	p.nextToken()

	if !p.curTokenKindIs(lexer.TokenCurlyBraceOpen) {
		p.add(p.error(p.curToken, "expected curly brace open {, instead got ", p.curToken.Text))
		return nil
	}

	p.nextToken()

	if p.curTokenKindIs(lexer.TokenBracketClose) {
		p.nextToken()
		return &ast.EnumExpression{
			Token: expr.Token,
			Body:  []*ast.AssignExpression{},
		}
	}

	body, err := p.parseEnumFields()

	if err != nil {
		p.add(err)
		return nil
	}

	expr.Body = body

	return expr
}

func (p *Parser) parseEnumFields() ([]*ast.AssignExpression, error) {
	fields := make([]*ast.AssignExpression, 0)

	assignExpr := &ast.AssignExpression{Token: p.curToken}

	for !p.curTokenKindIs(lexer.TokenCurlyBraceClose) {
		if !p.curTokenKindIs(lexer.TokenIdentifier) {
			err := p.error(p.curToken, "expected an identifier, instead got ", p.curToken.Text)
			p.syncUntilTokenIs(lexer.TokenCurlyBraceClose, true)
			return nil, err
		}

		field := p.parseIdentifier()

		assignExpr.Left = append(assignExpr.Left, field)

		// support for custom associated value
		if p.curTokenKindIs(lexer.TokenAssign) {
			// consume =
			p.nextToken()

			if !p.curTokenKindIs(lexer.TokenInt) {
				err := p.error(p.curToken, "expected an int literal, instead got ", p.curToken.Text)
				p.syncUntilTokenIs(lexer.TokenCurlyBraceClose, true)
				return nil, err
			}

			// associated value
			assignExpr.Right = append(assignExpr.Right, p.parseIntLiteral())
		}

		fields = append(fields, assignExpr)

		if !p.curTokenKindIs(lexer.TokenComma) {
			err := p.error(p.curToken, "expected a comma (,) at the end, instead got ", p.curToken.Text)
			p.syncUntilTokenIs(lexer.TokenCurlyBraceClose, true)
			return nil, err
		}
		p.nextToken()
	}

	p.nextToken()

	return fields, nil
}

func (p *Parser) parseWhileStatement() (*ast.WhileStatement, error) {
	stmt := &ast.WhileStatement{Token: p.curToken}
	p.nextToken()

	stmt.Condition = p.parseExpression(ASSIGN)

	if !p.curTokenKindIs(lexer.TokenCurlyBraceOpen) {
		return nil, fmt.Errorf("expected curly brace open ( { ), got shit")
	}

	p.nextToken()

	stmt.Body = p.parseBlockStatement().(*ast.BlockStatement)
	return stmt, nil
}

func (p *Parser) parseForStatement() (*ast.ForStatement, error) {
	stmt := &ast.ForStatement{Token: p.curToken}

	p.nextToken()

	if !p.curTokenKindIs(lexer.TokenIdentifier) {
		return nil, p.error(p.curToken, "expected at least one identifier, instead got ", p.curToken.Text)
	}

	p.nextToken()

	stmt.Identifiers = append(stmt.Identifiers, p.parseIdentifier().(*ast.Identifier))

	if p.curTokenKindIs(lexer.TokenComma) {
		ident, ok := p.parseIdentifier().(*ast.Identifier)
		if !ok {
			return nil, p.error(p.curToken, "expected an identifier, got shit")
		}
		stmt.Identifiers = append(stmt.Identifiers, ident)
	}

	if !p.curTokenKindIs(lexer.TokenIn) {
		return nil, p.error(p.curToken, "expected in, got shit")
	}

	// look ahead and see if the pattern <number>..<number>
	if p.peekTokenKindIs(lexer.TokenRange) {
		// use the range pattern struct fro the ast (ast.RangePattern)
		pattern := &ast.RangePattern{Token: p.curToken}
		pattern.Start = p.parseExpression(OR)

		if !p.curTokenKindIs(lexer.TokenRange) {
			return nil, p.error(p.curToken, "expected .. token, instead got ", p.curToken.Text)
		}

		// if operator exists it's only assign (=)
		if p.curTokenKindIs(lexer.TokenAssign) {
			pattern.Op = p.curToken.Text
			p.nextToken() // consume the operator
		} else {
			if _, ok := lexer.BinOperators[p.curToken.Kind]; ok {
				return nil, p.error(p.curToken, "only allowed operator is =, instead got ", p.curToken.Text)
			}
		}

		pattern.End = p.parseExpression(OR)
		stmt.Target = pattern
	} else {
		stmt.Target = p.parseExpression(OR)
	}

	if !p.curTokenKindIs(lexer.TokenCurlyBraceOpen) {
		return nil, p.error(p.curToken, "expected curly brace open { , instead got ", p.curToken.Text)
	}

	p.nextToken()

	stmt.Body = p.parseBlockStatement().(*ast.BlockStatement)
	return stmt, nil
}

func (p *Parser) parseNextStatement() (*ast.NextStatement, error) {
	stmt := &ast.NextStatement{Token: p.curToken}
	// consume the next token
	p.nextToken()
	return stmt, nil
}

func (p *Parser) parseBreakStatement() (*ast.BreakStatement, error) {
	stmt := &ast.BreakStatement{Token: p.curToken}
	// consume the break token
	p.nextToken()
	return stmt, nil
}

func (p *Parser) parseAssignStatement() (*ast.AssignStatement, error) {
	stmt := &ast.AssignStatement{Token: p.curToken}
	p.nextToken()

	stmt.Left = p.parsePrefixExpressionWrapper()

	// check for the token assign
	if !p.curTokenKindIs(lexer.TokenAssign) {
		return nil, p.error(p.curToken, "expected assign token (=), got shit")
	}

	stmt.Right = p.parsePrefixExpressionWrapper()

	return stmt, nil
}

func (p *Parser) parsePrefixExpressionWrapper() []ast.Expression {
	exps := make([]ast.Expression, 0)

	exps = append(exps, p.parseExpression(LOWEST))

	for p.curTokenKindIs(lexer.TokenComma) {
		// consume the comma (,) token
		p.nextToken()
		exps = append(exps, p.parseExpression(LOWEST))
	}

	return exps
}

func (p *Parser) parseIdentifier() ast.Expression {
	ident := &ast.Identifier{Token: p.curToken, Value: p.curToken.Text}
	p.nextToken()
	return ident
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

	for p.curTokenKindIs(lexer.TokenComma) {
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
	tok := p.curToken
	p.nextToken()

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
	tok := p.curToken
	p.nextToken()

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
	tok := p.curToken
	p.nextToken()

	return &ast.StringLiteral{
		Token: tok,
		Value: tok.Text,
	}
}

func (p *Parser) parseCharLiteral() ast.Expression {
	tok := p.curToken
	p.nextToken()

	code, _, _, err := strconv.UnquoteChar(tok.Text, '\'')
	if err != nil {
		p.add(p.error(tok, err.Error()))
		return nil
	}
	return &ast.CharLiteral{
		Token: tok,
		Value: code,
	}
}

func (p *Parser) parseNulLiteral() ast.Expression {
	tok := p.curToken
	p.nextToken()

	return &ast.NulLiteral{
		Token: tok,
	}
}

func (p *Parser) parseBooleanLiteral() ast.Expression {
	tok := p.curToken
	p.nextToken()

	truth := tok.Text == "true"
	return &ast.BooleanLiteral{
		Token: tok,
		Value: truth,
	}
}

func (p *Parser) parseArrayLiteral() ast.Expression {
	expr := &ast.ArrayLiteral{Token: p.curToken}

	if !p.curTokenKindIs(lexer.TokenBracketOpen) {
		p.add(p.error(expr.Token, "expected open bracket [, instead got ", p.curToken.Text))
		return nil
	}

	// consume the [
	p.nextToken()

	elements := make([]ast.Expression, 0)

	if p.curTokenKindIs(lexer.TokenBracketClose) {
		p.nextToken()
		return &ast.ArrayLiteral{
			Token:    expr.Token,
			Elements: elements,
		}
	}

	val := p.parseExpression(LOWEST)

	if val == nil {
		p.syncUntilTokenIs(lexer.TokenComma, false)
		goto round
	}

	elements = append(elements, val)

round:
	for p.curTokenKindIs(lexer.TokenComma) {
		p.nextToken()

		val := p.parseExpression(LOWEST)

		if val == nil {
			p.syncUntilTokenIs(lexer.TokenComma, false)
			goto round
		}

		elements = append(elements, val)
	}

	expr.Elements = elements

	if !p.curTokenKindIs(lexer.TokenBracketClose) {
		p.add(p.error(p.curToken, "expected close bracket ( ] ), instead got ", p.curToken.Text))
		return nil
	}

	// consume [
	p.nextToken()

	return expr
}

func (p *Parser) parseScope() (*ast.ScopeStatement, error) {
	stmt := &ast.ScopeStatement{Token: p.curToken}

	p.nextToken()

	stmt.Body = p.parseBlockStatement().(*ast.BlockStatement)
	return stmt, nil
}

func (p *Parser) parseMapLiteral() ast.Expression {
	prev := p.curToken

	if !p.curTokenKindIs(lexer.TokenCurlyBraceOpen) {
		p.add(p.error(prev, "expected open curly brace {, instead got ", p.curToken.Text))
	}

	p.nextToken()

	pairs := make(map[ast.Expression]ast.Expression, 0)

round:
	for !p.curTokenKindIs(lexer.TokenCurlyBraceClose) {
		key := p.parseExpression(LOWEST)

		if key == nil {
			p.sync(false)
			goto round
		}

		if !p.curTokenKindIs(lexer.TokenColon) {
			p.add(p.error(p.curToken, "expected colon : after key, instead got ", p.curToken.Text))
			p.sync(false)
			goto round
		}

		// consume :
		p.nextToken()

		value := p.parseExpression(LOWEST)

		if value == nil {
			p.sync(false)
			goto round
		}

		pairs[key] = value

		if !p.curTokenKindIs(lexer.TokenComma) {
			p.add(p.error(p.curToken, "expected comma (,) at the end, instead got ", p.curToken.Text))
			return nil
		}

		p.nextToken()
	}

	if !p.curTokenKindIs(lexer.TokenCurlyBraceClose) {
		p.add(p.error(p.curToken, "expected close bracket ( ] ), instead got ", p.curToken.Text))
		return nil
	}

	// consume }
	p.nextToken()

	return &ast.MapLiteral{
		Token: prev,
		Pairs: pairs,
	}
}

func (p *Parser) parseGroupedExpression() ast.Expression {
	p.nextToken()
	exp := p.parseExpression(LOWEST)
	if !p.curTokenKindIs(lexer.TokenBraceClose) {
		return nil
	}
	return exp
}

func (p *Parser) parseIfExpression() ast.Expression {
	expr := &ast.IfExpression{Token: p.curToken}
	p.nextToken()

	// this is to prevent the launch of parse struct instance func
	p.internalFlags = append(p.internalFlags, "if-mode")
	expr.Condition = p.parseExpression(ASSIGN)
	p.internalFlags = slices.DeleteFunc(p.internalFlags, func(elem string) bool {
		return elem == "if-mode"
	})

	// look ahead to the next token
	if p.curTokenKindIs(lexer.TokenQuestion) || p.curTokenKindIs(lexer.TokenUse) {
		p.nextToken() // consume the ?

		exprStmt, err := p.parseExpressionStatement()
		if err != nil {
			return nil
		}
		expr.Consequence = &ast.BlockStatement{
			Body: []ast.Statement{exprStmt},
		}

		p.nextToken()
		if !p.curTokenKindIs(lexer.TokenColon) && !p.curTokenKindIs(lexer.TokenElse) {
			p.add(p.error(p.curToken, "expected else or : as following token for the ternary definition, got ", p.curToken.Kind))
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
		if !p.curTokenKindIs(lexer.TokenCurlyBraceOpen) {
			p.add(p.error(p.curToken, "expected close curly brace ( } ), instead got ", p.curToken.Text))
			return nil
		}

		p.nextToken()

		expr.Consequence = p.parseBlockStatement().(*ast.BlockStatement)

		// check if there is an else stmt
		if p.curTokenKindIs(lexer.TokenElse) {
			p.nextToken()
			// support for else if
			if p.curTokenKindIs(lexer.TokenIf) {
				expr.Alternative = p.parseIfExpression()
			} else {
				if !p.curTokenKindIs(lexer.TokenCurlyBraceOpen) {
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

func (p *Parser) parseFunctionExpression() ast.Expression {
	expr := &ast.FunctionExpression{Token: p.curToken}
	p.nextToken()

	if !p.curTokenKindIs(lexer.TokenBraceOpen) {
		p.add(p.error(p.curToken, "expected brace open '(' ,instead got ", p.curToken.Text))
		return nil
	}

	p.nextToken()

	self, args := p.parseArguments()

	// isn't required to exist
	expr.Self = self
	expr.Args = args

	if !p.curTokenKindIs(lexer.TokenCurlyBraceOpen) {
		p.add(p.error(p.curToken, "expected curly brace open ( { ), instead got ", p.curToken.Text))
		return nil
	}

	p.nextToken()

	body := p.parseBlockStatement().(*ast.BlockStatement)

	if body == nil {
		p.add(p.error(p.curToken, "expected valid body, instead got ", p.curToken.Text))
		return nil
	}

	expr.Body = body

	return expr
}

func (p *Parser) parseArguments() (*ast.Identifier, []*ast.Arg) {
	// return another identifier which is
	args := make([]*ast.Arg, 0)
	self := &ast.Identifier{}

	// self needs to be defined at first
	if p.curTokenKindIs(lexer.TokenSelf) {
		self.Token = p.curToken
		self.Value = p.curToken.Text
		// consume the self
		p.nextToken()
		if p.curTokenKindIs(lexer.TokenComma) {
			// consume the comma
			p.nextToken()
		}
	}

	if p.curTokenKindIs(lexer.TokenBraceClose) {
		p.nextToken()
		return self, args
	}

	arg := &ast.Arg{
		Token: p.curToken,
		Name: &ast.Identifier{
			Token: p.curToken,
			Value: p.curToken.Text,
		},
	}

	p.nextToken()
	// expect colon
	if !p.curTokenKindIs(lexer.TokenColon) {
		p.add(p.error(p.curToken, "expected : after argument name, instead got ", p.curToken.Text))
		return nil, nil
	}

	p.nextToken()

	// parse type
	tp, err := p.parseType()

	if err != nil {
		p.add(err)
		return nil, nil
	}

	arg.Type = tp

	args = append(args, arg)

	for p.curTokenKindIs(lexer.TokenComma) {
		p.nextToken()
		arg := &ast.Arg{
			Token: p.curToken,
			Name: &ast.Identifier{
				Token: p.curToken,
				Value: p.curToken.Text,
			},
		}

		// expect colon
		if !p.curTokenKindIs(lexer.TokenColon) {
			p.add(p.error(p.curToken, "expected : after argument name, instead got ", p.curToken.Text))
			return nil, nil
		}

		p.nextToken()

		// parse type
		tp, err := p.parseType()

		if err != nil {
			p.add(err)
			return nil, nil
		}

		arg.Type = tp

		args = append(args, arg)
	}

	if !p.curTokenKindIs(lexer.TokenBraceClose) {
		p.add(p.error(p.curToken, "expected ) token in function definition, instead got ", p.curToken.Text))
		return nil, nil
	}

	p.nextToken()

	return self, args
}

func (p *Parser) parseBlockStatement() ast.Expression {
	block := ast.BlockStatement{Token: p.curToken}
	block.Body = make([]ast.Statement, 0)

	for !p.curTokenKindIs(lexer.TokenCurlyBraceClose) && !p.curTokenKindIs(lexer.TokenEOF) {
		// parse body expressions and statements
		stmt, err := p.parseStatement()

		if err != nil {
			p.add(err)
			p.sync(false)
		} else {
			block.Body = append(block.Body, stmt)
		}
	}

	if !p.curTokenKindIs(lexer.TokenCurlyBraceClose) {
		p.error(p.curToken, "end of block expression expects }, instead got ", p.curToken.Text)
		return nil
	}

	p.nextToken()

	return &block
}

func (p *Parser) parseCallExpression(left ast.Expression) ast.Expression {
	switch left.(type) {
	case *ast.Identifier:
	default:
		p.add(p.error(p.curToken, "only call are allowed, bounding function into a variable ain't allowed"))
		return nil
	}

	exp := ast.CallExpression{Token: left.GetToken(), Function: *(left.(*ast.Identifier))}

	exp.Args = p.parseCallArguments()

	return &exp
}

func (p *Parser) parseCallArguments() []ast.Expression {
	args := make([]ast.Expression, 0)

	if !p.curTokenKindIs(lexer.TokenBraceOpen) {
		p.add(p.error(p.curToken, "expect brace (, instead got ", p.curToken.Text))
		return nil
	}

	p.nextToken()

	if p.curTokenKindIs(lexer.TokenBraceClose) {
		p.nextToken()
		return args
	}

	args = append(args, p.parseExpression(LOWEST))

	for p.curTokenKindIs(lexer.TokenComma) {
		p.nextToken()
		expr := p.parseExpression(LOWEST)
		if expr == nil {
			return nil
		}
		args = append(args, expr)
	}

	if !p.curTokenKindIs(lexer.TokenBraceClose) {
		p.add(p.error(p.curToken, "expect brace ), instead got ", p.curToken.Text))
		return nil
	}

	p.nextToken()

	return args
}

func (p *Parser) parseIndexExpression(left ast.Expression) ast.Expression {
	exp := &ast.IndexExpression{Token: left.GetToken(), Left: left}
	p.nextToken()

	switch {
	case p.curTokenKindIs(lexer.TokenColon):
		exp.Start = p.parseExpression(LOWEST)

	case p.curTokenKindIs(lexer.TokenColon):
		exp.Range = true
		p.nextToken() // consume :

	case p.curTokenKindIs(lexer.TokenBracketClose):
		exp.End = p.parseExpression(LOWEST)
	}

	if !p.curTokenKindIs(lexer.TokenBracketClose) {
		p.add(p.error(p.curToken, "expect brace ], instead got ", p.curToken.Text))
		return nil
	}

	p.nextToken()

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
	fields, err := p.parseFieldValues()

	if err != nil {
		p.add(err)
		return nil
	}

	expr.Body = fields

	return expr
}

func (p *Parser) parseFieldValues() ([]ast.FieldInstance, error) {
	fields := make([]ast.FieldInstance, 0)
	p.nextToken()

	if p.curTokenKindIs(lexer.TokenCurlyBraceClose) {
		p.nextToken()
		return fields, nil
	}

	identifier, ok := p.parseIdentifier().(*ast.Identifier)

	if !ok {
		return []ast.FieldInstance{}, p.error(p.curToken, "expected an identifier, got ", p.curToken.Text)
	}

	if !p.curTokenKindIs(lexer.TokenColon) {
		return []ast.FieldInstance{}, p.error(p.curToken, "expected : after identifier, instead got ", p.curToken.Text)
	}

	p.nextToken()

	value := p.parseExpression(LOWEST)

	fields = append(fields, ast.FieldInstance{
		Key:   identifier,
		Value: value,
	})

	for p.curTokenKindIs(lexer.TokenComma) {
		p.nextToken()
		identifier, ok := p.parseIdentifier().(*ast.Identifier)

		if !ok {
			return fields, p.error(p.curToken, "expected an identifier, got ", p.curToken)
		}

		if !p.curTokenKindIs(lexer.TokenColon) {
			return fields, p.error(p.curToken, "expected : after identifier, instead got ", p.curToken.Text)
		}

		p.nextToken()

		value := p.parseExpression(LOWEST)

		fields = append(fields, ast.FieldInstance{
			Key:   identifier,
			Value: value,
		})
	}

	if !p.curTokenKindIs(lexer.TokenCurlyBraceClose) {
		return fields, p.error(p.curToken, "expect curly brace close }, instead got ", p.curToken.Text)
	}

	p.nextToken()

	return fields, nil
}

func (p *Parser) parseMemberShipAccess(left ast.Expression) ast.Expression {
	expr := &ast.MemberShipExpression{Token: left.GetToken(), Object: left}

	if !p.curTokenKindIs(lexer.TokenDot) {
		p.add(p.error(p.curToken, "expect dot token (.), instead got ", p.curToken.Text))
		return nil
	}

	p.nextToken()

	// the precedence needs to be >= () function call
	expr.Property = p.parseExpression(PREFIX)
	// ? if these results in more bugs consider changing it to a binary expression where the operator is a .
	// then in the evaluation layer we see what operator is it, and then do something

	return expr
}

// this function is responsible to parsing the assign operator syntax
// an example of this: index += 1 <=> index = index + 1
func (p *Parser) parseAssignOperatorExpression(left ast.Expression) ast.Expression {
	expr := &ast.AssignExpression{Token: left.GetToken(), Left: []ast.Expression{left}}

	// get the operator, from the current op which can be something (+=,%=,..etc)
	operator := strings.Split(p.curToken.Text, "=")[0]
	// consume the operator token
	p.nextToken()
	// parse the operator
	expr.Right = []ast.Expression{
		&ast.BinaryExpression{
			Token:    p.curToken,
			Operator: operator,
			Left:     expr.Left[0],
			Right:    p.parseExpression(LOWEST),
		},
	}

	return expr
}

// this function is responsible of parsing the double operator assign
// an example of this : index++, index-- <=> index = index + 1
// only support for (+,-) operators
func (p *Parser) parseDoubleOperatorExpression(left ast.Expression) ast.Expression {
	expr := &ast.AssignExpression{Token: left.GetToken(), Left: []ast.Expression{left}}

	operator := string(p.curToken.Text[0])

	// parse the operator
	expr.Right = []ast.Expression{
		&ast.BinaryExpression{
			Token:    p.curToken,
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

	return expr
}

func (p *Parser) parseBindExpression() (ast.Statement, error) {
	stmt := &ast.VarDeclaration{Token: lexer.Token{
		LiteralToken: lexer.LiteralToken{
			Text: "let",
			Kind: lexer.TokenLet,
		},
		Col: p.curToken.Col,
		Row: p.curToken.Row,
	}, Mutable: true}

	stmt.Name = p.parseIdentifiers()

	switch p.curToken.Kind {
	case lexer.TokenBind:
		stmt.Token = lexer.Token{
			LiteralToken: lexer.LiteralToken{
				Text: "const",
				Kind: lexer.TokenConst,
			},
		}
		stmt.Mutable = false
		p.nextToken()

	case lexer.TokenWalrus:
		p.nextToken()

	case lexer.TokenColon:
		// consume the :

		tp, err := p.parseType()
		if err != nil {
			return nil, err
		}

		stmt.Type = tp

		if !p.curTokenKindIs(lexer.TokenAssign) && !p.curTokenKindIs(lexer.TokenColon) {
			return nil, p.error(p.curToken, "expected assign (= | :) after", tp, " token got ", p.curToken.Text)
		}

		if p.curTokenKindIs(lexer.TokenColon) {
			stmt.Token = lexer.Token{
				LiteralToken: lexer.LiteralToken{
					Text: "const",
					Kind: lexer.TokenConst,
				},
			}
			stmt.Mutable = false
		}

	default:
		return nil, p.error(p.curToken, "expected (:= or ::) operators, instead got ", p.curToken.Text)
	}

	stmt.Value = p.parseExpression(LOWEST)

	return stmt, nil
}

func (p *Parser) parsePrefixExpression() ast.Expression {
	tok := p.curToken

	if _, ok := lexer.UnaryOperators[tok.Kind]; !ok {
		p.add(p.error(tok, "expected a unary operator (! | - | ~), instead got ", p.curToken.Text))
		return nil
	}

	p.nextToken()

	right := p.parseExpression(PREFIX)

	return &ast.UnaryExpression{
		Token:    tok,
		Operator: tok.Text,
		Right:    right,
	}
}

func (p *Parser) parseInfixExpression(left ast.Expression) ast.Expression {
	tok := p.curToken

	if _, ok := lexer.BinOperators[tok.Kind]; !ok {
		p.add(p.error(tok, "expected a binary operator (== | > | < | ...), instead got ", tok.Text))
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
	cur := p.curToken

	if cur.Kind == lexer.TokenError {
		p.add(p.error(cur, cur.Text))
		return nil
	}

	prefix := p.prefixParseFns[cur.Kind]

	if prefix == nil {
		p.add(p.error(cur, "unrecognized token ", cur.Text))
		return nil
	}

	leftExp := prefix()
	cur = p.curToken

	if cur.Kind == lexer.TokenBraceOpen {
		// make sure that the token before is an identifier
		lookBeforeKind := p.prevToken.Kind
		_, ok := lexer.BinOperators[lookBeforeKind]
		if lookBeforeKind != lexer.TokenIdentifier && !ok && cur.Col > 1 {
			p.add(p.error(p.curToken, "brace token expects to be an identifier before it, or a binary operator"))
			return nil
		}
	}

	for p.curToken.Row <= cur.Row && p.curToken.Kind != lexer.TokenEOF && precedence < p.peekPrecedence() && p.prevToken.Row == cur.Row {
		infix := p.infixParseFns[p.curToken.Kind]
		if infix == nil {
			return leftExp
		}
		leftExp = infix(leftExp)
	}

	return leftExp
}
