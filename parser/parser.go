package parser

import (
	"blk/ast"
	"blk/lexer"
	"errors"
	"fmt"
	"path/filepath"
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
	lexer.TokenAssignMinus:         SUM,
	lexer.TokenMinus:               SUM,
	lexer.TokenSlash:               PRODUCT,
	lexer.TokenAssignSlash:         PRODUCT,
	lexer.TokenMultiply:            PRODUCT,
	lexer.TokenAssignMultiply:      PRODUCT,
	lexer.TokenModule:              PRODUCT,
	lexer.TokenAssignModule:        PRODUCT,
	lexer.TokenExclamation:         PREFIX,
	lexer.TokenBitNot:              PREFIX,
	lexer.TokenAssignPlusOne:       PREFIX,
	lexer.TokenAssignMinusOne:      PREFIX,
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
	filePath       string
	fileName       string
	errors         []error
	prefixParseFns map[lexer.TokenKind]prefixParseFn
	infixParseFns  map[lexer.TokenKind]infixParseFn
	internalFlags  []string

	prevToken lexer.Token // previous token of current token
	curToken  lexer.Token
	peekToken lexer.Token // one token lookahead

	directives map[string]lexer.Token // for directives that are not tied in a special place
}

func NewParser(lex *lexer.Lexer, filepath, filename string) *Parser {
	p := Parser{
		lexer:          lex,
		filePath:       filepath,
		fileName:       filename,
		errors:         []error{},
		prefixParseFns: make(map[lexer.TokenKind]prefixParseFn),
		infixParseFns:  make(map[lexer.TokenKind]infixParseFn),
		internalFlags:  []string{},
		directives:     make(map[string]lexer.Token),
	}

	// prefix/unary operators
	p.registerPrefix(lexer.TokenIdentifier, p.parseIdentifier)
	p.registerPrefix(lexer.TokenLine, p.parseDirectiveToValue)
	p.registerPrefix(lexer.TokenFile, p.parseDirectiveToValue)
	p.registerPrefix(lexer.TokenDir, p.parseDirectiveToValue)
	p.registerPrefix(lexer.TokenInteger, p.parseIntLiteral)
	p.registerPrefix(lexer.TokenFloat, p.parseFloatLiteral)
	p.registerPrefix(lexer.TokenStr, p.parseStringLiteral)
	p.registerPrefix(lexer.TokenChar, p.parseCharLiteral)
	p.registerPrefix(lexer.TokenNul, p.parseNulLiteral)
	p.registerPrefix(lexer.TokenBracketOpen, p.parseArrayLiteral)
	p.registerPrefix(lexer.TokenMap, p.parseMapLiteral)
	p.registerPrefix(lexer.TokenExclamation, p.parsePrefixExpression)
	p.registerPrefix(lexer.TokenBitNot, p.parsePrefixExpression)
	p.registerPrefix(lexer.TokenMinus, p.parsePrefixExpression)
	p.registerPrefix(lexer.TokenMultiply, p.parsePrefixExpression)
	p.registerPrefix(lexer.TokenBitAnd, p.parsePrefixExpression)
	p.registerPrefix(lexer.TokenBool, p.parseBooleanLiteral)
	p.registerPrefix(lexer.TokenBraceOpen, p.parseGroupedExpression)
	p.registerPrefix(lexer.TokenIf, p.parseIfExpression)
	p.registerPrefix(lexer.TokenFn, p.parseFunctionExpression)
	p.registerPrefix(lexer.TokenSwitch, p.parseSwitchExpression)
	p.registerPrefix(lexer.TokenCast, p.parseCastExpression)
	p.registerPrefix(lexer.TokenDot, p.parseNonExplicitStructInstanceExpression)

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
	// ? Double operators: ++, --
	p.registerInfix(lexer.TokenAssignPlusOne, p.parseDoubleOperatorExpression)
	p.registerInfix(lexer.TokenAssignMinusOne, p.parseDoubleOperatorExpression)
	// ? Arithmetic assign operator : add +=, -=, *=, /=, %=, &=, |=, ^=, <<=, >>=, &&=, ||=
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

func (p *Parser) GetErrors() []error {
	return p.errors
}

func (p *Parser) peekPrecedence() int {
	if p, ok := precedences[p.curToken.Kind]; ok {
		return p
	}
	return LOWEST
}

func (p *Parser) nextToken() {
	p.prevToken = p.curToken
	p.curToken = p.peekToken
	p.peekToken = p.lexer.NextToken()

	// consume comment tokens
	for p.curTokenKindIs(lexer.TokenComment) {
		p.nextToken()
	}
}

func (p *Parser) add(err error) {
	if len(err.Error()) > 0 {
		p.errors = append(p.errors, err)
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

func (p *Parser) curTokenKindIs(kind lexer.TokenKind) bool {
	return p.curToken.Kind == kind
}

func (p *Parser) peekTokenKindIs(kind lexer.TokenKind) bool {
	return p.peekToken.Kind == kind
}

func (p *Parser) error(tok lexer.Token, msg ...interface{}) error {
	errMsg := fmt.Sprintf("\033[1;90m%s:%d:%d:\033[0m ERROR: %s", p.fileName, tok.Row, tok.Col, fmt.Sprint(msg...))

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
		} else if stmt != nil {
			ast.Statements = append(ast.Statements, stmt)
		}
	}

	return &ast
}

// TODO: better error handling and targeting

func (p *Parser) parseStatement() (ast.Statement, error) {
	switch p.curToken.Kind {
	case lexer.TokenComment:
		return p.parseCommentStatement()
	case lexer.TokenLet, lexer.TokenConst:
		return p.parseDeclaration()
	case lexer.TokenType:
		return p.parseTypeStatement()
	case lexer.TokenTest:
		return p.parseTestStatement()
	case lexer.TokenReturn:
		return p.parseReturnStatement()
	case lexer.TokenImport:
		return p.parseImportStatement()
	case lexer.TokenFor:
		return p.parseForStatement()
	case lexer.TokenNext:
		return p.parseNextStatement()
	case lexer.TokenBreak:
		return p.parseBreakStatement()
	case lexer.TokenCurlyBraceOpen:
		return p.parseScope()
	case lexer.TokenDeprecated:
		// check there is a reason
		p.nextToken()
		if !p.curTokenKindIs(lexer.TokenStr) {
			return nil, p.error(p.curToken, "expected a string after #deprecated directive, instead got ", p.curToken.Text)
		}

		p.directives["#deprecated"] = p.curToken
		p.nextToken()

		return nil, nil

	case lexer.TokenScope:
		p.nextToken()
		if !p.curTokenKindIs(lexer.TokenStr) {
			return nil, p.error(p.curToken, "expected a string after #deprecated directive, instead got ", p.curToken.Text)
		}

		p.directives["#scope"] = p.curToken
		p.nextToken()

		return nil, nil

	case lexer.TokenIdentifier:

		if p.peekTokenKindIs(lexer.TokenComma) || p.peekTokenKindIs(lexer.TokenAssign) {
			return p.parseMultiAssignStatement()
		}

		return p.parseExpressionStatement()

	case lexer.TokenError:
		return nil, p.error(p.curToken, p.curToken.Text)

	default:
		return p.parseExpressionStatement()
	}
}

func (p *Parser) parseCompositeType(prev lexer.Token) (*ast.CompositeType, error) {
	// composite type
	tp := &ast.CompositeType{Token: prev}
	if prev.Kind == lexer.TokenBracketOpen {
		tp.Kind = ast.TypeArray
	} else {
		tp.Kind = ast.TypeMap
		p.nextToken()
	}

	if !p.curTokenKindIs(lexer.TokenBracketOpen) {
		return nil, p.error(p.curToken, "expected [ after ", p.curToken.Kind, " instead got ", p.curToken.Text)
	}

	// consume the ( token
	p.nextToken()

	if tp.Kind == ast.TypeArray {
		// parse the size
		if p.curTokenKindIs(lexer.TokenRange) {

			tp.Size = &ast.IntegerLiteral{
				Token: p.curToken,
				Value: -1,
			}

			p.nextToken()

		} else {
			if !p.curTokenKindIs(lexer.TokenInteger) {
				return nil, p.error(p.curToken, "expected an int literal | identifier after ", tp.LeftType, " instead got ", p.curToken.Text)
			}

			tp.Size = p.parseIntLiteral().(*ast.IntegerLiteral)
		}
	} else {

		if _, ok := lexer.TypeKeywords[p.curToken.Kind]; !ok && !p.curTokenKindIs(lexer.TokenBracketOpen) {
			return nil, p.error(p.curToken, "expected a type, but instead got ", p.curToken.Text)
		}

		// call the parse type on the current token
		nestedType, err := p.parseType()
		if err != nil {
			return nil, err
		}
		tp.LeftType = nestedType
	}

	// check the end token is )
	if !p.curTokenKindIs(lexer.TokenBracketClose) {
		return nil, p.error(p.curToken, "after value type ", tp.RightType, " a ) is expected, instead got ", p.curToken.Text)
	}

	// consume ]
	p.nextToken()

	if _, ok := lexer.TypeKeywords[p.curToken.Kind]; !ok && !p.curTokenKindIs(lexer.TokenBracketOpen) {
		return nil, p.error(p.curToken, "expected a type, but instead got ", p.curToken.Text)
	}

	// parse the second type
	nestedType, err := p.parseType()
	if err != nil {
		return nil, err
	}

	if tp.Kind == ast.TypeMap {
		tp.RightType = nestedType
	} else {
		tp.LeftType = nestedType
	}

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

	if !p.curTokenKindIs(lexer.TokenArrow) {
		return nil, p.error(p.curToken, "expected -> after ) instead got ", p.curToken.Text)
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

func (p *Parser) parseTypeAlias(tok lexer.Token) (ast.Type, error) {
	alias := &ast.AliasType{Token: tok, Kind: ast.TypeAlias}
	alias.Alias = p.parseIdentifier().(*ast.Identifier)
	return alias, nil
}

func (p *Parser) parsePrimitiveType(tok lexer.Token) (ast.Type, error) {
	primitive := &ast.PrimitiveType{
		Token: tok,
	}
	// parse primitive type

	switch tok.Kind {
	case lexer.TokenAny:
	case lexer.TokenBool, lexer.TokenString, lexer.TokenChar:

	case lexer.TokenSInt8, lexer.TokenSInt16, lexer.TokenSInt32, lexer.TokenSInt64:
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

	primitive.Kind = ast.PrimitiveTypes[primitive.Token.Kind]

	p.nextToken()

	return primitive, nil
}

// parses explicit types
func (p *Parser) parseType() (ast.Type, error) {
	// current token

	switch p.curToken.Kind {
	case lexer.TokenBracketOpen, lexer.TokenMap:
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
		if exp := p.parseStructType(); exp != nil {
			return exp, nil
		}
		return nil, nil

	case lexer.TokenEnum:
		if exp := p.parseEnumType(); exp != nil {
			return exp, nil
		}
		return nil, nil

	case lexer.TokenUnion:
		if exp := p.parseUnionType(); exp != nil {
			return exp, nil
		}
		return nil, nil

	case lexer.TokenIdentifier:
		return p.parseTypeAlias(p.curToken)

	default:
		// primitive type
		return p.parsePrimitiveType(p.curToken)
	}

}

func (p *Parser) addDirective(stmt *ast.Declaration) {
	if !p.curTokenKindIs(lexer.TokenLine) && !p.curTokenKindIs(lexer.TokenFile) && !p.curTokenKindIs(lexer.TokenDir) {
		_, curIsDirective := lexer.Directives[p.curToken.Kind]
		for curIsDirective {
			if p.curTokenKindIs(lexer.TokenInline) {
				stmt.Inline = true
			} else {
				stmt.Directive[p.curToken.Text] = &ast.StringLiteral{
					Token: p.curToken,
					Value: p.curToken.Text,
				}
			}
			p.nextToken()
			_, curIsDirective = lexer.Directives[p.curToken.Kind]
		}
	}
}

func (p *Parser) parseDeclaration() (*ast.Declaration, error) {
	stmt := &ast.Declaration{Token: p.curToken, Directive: make(map[string]*ast.StringLiteral)}

	for directive, dValue := range p.directives {
		stmt.Directive[directive] = &ast.StringLiteral{Token: dValue, Value: dValue.Text}
	}

	// reset map
	clear(p.directives)

	stmt.Mutable = stmt.Token.Kind == lexer.TokenLet

	p.nextToken()

	stmt.Name = p.parseIdentifiers()

	if p.curTokenKindIs(lexer.TokenColon) {
		// consume :
		p.nextToken()

		// type
		tp, err := p.parseType()

		if err != nil {
			return nil, err
		}

		stmt.Type = tp
	}

	if !p.curTokenKindIs(lexer.TokenAssign) && p.curToken.Row == p.prevToken.Row {
		return nil, p.error(p.curToken, "expected assign (=), got ", p.curToken.Text)
	}

	if p.curTokenKindIs(lexer.TokenAssign) {
		// consume =
		p.nextToken()

		if p.curTokenKindIs(lexer.TokenNoInit) {
			stmt.Directive[lexer.TokenNoInit] = &ast.StringLiteral{
				Token: p.curToken,
				Value: "no explicit init",
			}

			p.nextToken()
			return stmt, nil
		}

		// collect all directives
		p.addDirective(stmt)

		exprs := append([]ast.Expression{}, p.parseExpression(LOWEST))

		for p.curTokenKindIs(lexer.TokenComma) {
			p.nextToken() // eat comma
			exprs = append(exprs, p.parseExpression(LOWEST))
		}

		stmt.Value = exprs
	}

	return stmt, nil
}

func (p *Parser) parseTypeStatement() (*ast.TypeDeclaration, error) {
	stmt := &ast.TypeDeclaration{Token: p.curToken, Directive: make(map[string]*ast.StringLiteral)}
	p.nextToken()

	for directive, dValue := range p.directives {
		stmt.Directive[directive] = &ast.StringLiteral{Token: dValue, Value: dValue.Text}
	}

	// reset map
	clear(p.directives)

	if !p.curTokenKindIs(lexer.TokenIdentifier) {
		return nil, p.error(p.curToken, "expected an identifier, instead got ", p.curToken.Text)
	}

	stmt.Alias = p.parseIdentifier().(*ast.Identifier)

	if !p.curTokenKindIs(lexer.TokenAssign) {
		return nil, p.error(p.curToken, "expected an assign (=), instead got ", p.curToken.Text)
	}

	p.nextToken()

	if p.curTokenKindIs(lexer.TokenDistinct) {
		stmt.Distinct = true
		p.nextToken()
	}

	tp, err := p.parseType()

	if err != nil {
		return nil, err
	}

	stmt.Type = tp

	return stmt, nil
}

func (p *Parser) parseTestStatement() (*ast.TestStatement, error) {
	stmt := &ast.TestStatement{Token: p.curToken}
	p.nextToken()

	if !p.curTokenKindIs(lexer.TokenStr) {
		return nil, p.error(p.curToken, "expected a test name, instead got ", p.curToken.Text)
	}

	stmt.Name = p.curToken.Text
	p.nextToken()

	if !p.curTokenKindIs(lexer.TokenCurlyBraceOpen) {
		return nil, p.error(p.curToken, "expected a {, instead got ", p.curToken.Text)
	}
	p.nextToken()

	body := p.parseBlockStatement()

	if body == nil {
		return nil, nil
	}

	stmt.Body = body.(*ast.BlockExpression)

	return stmt, nil
}

func (p *Parser) parseReturnStatement() (*ast.ReturnStatement, error) {
	stmt := &ast.ReturnStatement{Token: p.curToken}
	p.nextToken()

	returnValues := make([]ast.Expression, 0)

	if p.curToken.Row > p.prevToken.Row {
		stmt.ReturnValues = returnValues
		return stmt, nil
	}

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

	if p.curTokenKindIs(lexer.TokenDot) {
		// means import to the current scope
		stmt.IntoScope = true
		p.nextToken()
	}

	if p.curTokenKindIs(lexer.TokenIdentifier) {
		// means this is an alias
		stmt.Alias = p.parseIdentifier().(*ast.Identifier)
	}

	if !p.curTokenKindIs(lexer.TokenStr) {
		return nil, p.error(p.curToken, "expected a string as module path, instead got ", p.curToken.Text)
	}

	stmt.ModuleName = p.parseStringLiteral().(*ast.StringLiteral)

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

func (p *Parser) parseStructType() *ast.StructType {
	expr := &ast.StructType{Token: p.curToken}
	p.nextToken()

	if !p.curTokenKindIs(lexer.TokenCurlyBraceOpen) {
		p.add(p.error(p.curToken, "expected curly brace open {, instead got ", p.curToken.Text))
		return nil
	}

	p.nextToken()

	if p.curTokenKindIs(lexer.TokenBracketClose) {
		p.nextToken()
		return &ast.StructType{
			Token:  expr.Token,
			Fields: []*ast.Declaration{},
		}
	}

	fields, err := p.parseFields()

	if err != nil {
		p.add(err)
		return nil
	}

	expr.Fields = fields

	return expr
}

func (p *Parser) parseFields() ([]*ast.Declaration, error) {
	fields := make([]*ast.Declaration, 0)

	// parse until, then consume it
	for !p.curTokenKindIs(lexer.TokenCurlyBraceClose) {

		if p.curTokenKindIs(lexer.TokenBake) {
			// for now skip it
			field := &ast.Declaration{Token: p.curToken, Mutable: true, Directive: map[string]*ast.StringLiteral{}}
			field.Directive["#bake"] = &ast.StringLiteral{Token: p.curToken, Value: "#bake"}

			p.nextToken()

			if !p.curTokenKindIs(lexer.TokenIdentifier) {
				err := p.error(p.curToken, "expected an identifier after #bake directive, instead got ", p.curToken.Text)
				return nil, err
			}

			ident := p.parseIdentifier().(*ast.Identifier)
			field.Name = append(field.Name, ident)

			fields = append(fields, field)

			// check if there is a comma
			if !p.curTokenKindIs(lexer.TokenComma) {
				err := p.error(p.curToken, "expected an comma (,) at the end of each field, instead got ", p.curToken.Text)
				return nil, err
			}

			p.nextToken()

		} else {

			if p.peekTokenKindIs(lexer.TokenColon) {
				// parse type
				field := &ast.Declaration{Token: p.peekToken, Mutable: true, Directive: make(map[string]*ast.StringLiteral)}

				if !p.curTokenKindIs(lexer.TokenIdentifier) {
					err := p.error(p.prevToken, "expected an identifier, got ", p.prevToken.Text)
					return nil, err
				}

				ident := p.parseIdentifier().(*ast.Identifier)

				field.Name = append(field.Name, ident)

				// consume the :
				p.nextToken()

				tp, err := p.parseType()

				if err != nil {
					return nil, err
				}

				field.Type = tp

				// check if there is a default value or not
				if p.curTokenKindIs(lexer.TokenAssign) {
					p.nextToken() // consume = token

					p.addDirective(field)

					if p.curTokenKindIs(lexer.TokenComma) {
						goto noExpr
					}

					val := p.parseExpression(LOWEST)

					if val == nil {
						return nil, fmt.Errorf("")
					}

					field.Value = []ast.Expression{val}
				}

			noExpr:
				fields = append(fields, field)

				// check if there is a comma
				if !p.curTokenKindIs(lexer.TokenComma) {
					err := p.error(p.curToken, "expected an comma (,) at the end of each field, instead got ", p.curToken.Text)
					p.add(err)
				}

				p.nextToken()

			} else {
				// throw an error here
				err := p.error(p.curToken, "expected either (:: or :), instead got ", p.curToken.Text)
				p.add(err)
				p.nextToken()
			}
		}
	}

	p.nextToken()

	return fields, nil
}

func (p *Parser) parseEnumType() *ast.EnumType {
	expr := &ast.EnumType{Token: p.curToken}

	// consume the enum lexer.token
	p.nextToken()

	if !p.curTokenKindIs(lexer.TokenCurlyBraceOpen) {
		p.add(p.error(p.curToken, "expected curly brace open {, instead got ", p.curToken.Text))
		return nil
	}

	p.nextToken()

	if p.curTokenKindIs(lexer.TokenBracketClose) {
		p.nextToken()
		return &ast.EnumType{
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

	for !p.curTokenKindIs(lexer.TokenCurlyBraceClose) {
		assignExpr := &ast.AssignExpression{Token: p.curToken, Directive: make(map[string]*ast.StringLiteral)}

		if p.curTokenKindIs(lexer.TokenBake) {
			// bake directive
			assignExpr.Directive["#bake"] = &ast.StringLiteral{Token: p.curToken, Value: "#bake"}

			p.nextToken()

			if !p.curTokenKindIs(lexer.TokenIdentifier) {
				err := p.error(p.curToken, "expected an identifier after #bake directive, instead got ", p.curToken.Text)
				return nil, err
			}

			ident := p.parseIdentifier().(*ast.Identifier)

			assignExpr.Embeddable = append(assignExpr.Embeddable, ident)
		} else {
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

				if !p.curTokenKindIs(lexer.TokenInteger) {
					err := p.error(p.curToken, "expected an int literal, instead got ", p.curToken.Text)
					p.syncUntilTokenIs(lexer.TokenCurlyBraceClose, true)
					return nil, err
				}

				// associated value
				assignExpr.Right = append(assignExpr.Right, p.parseIntLiteral())
			}
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

func (p *Parser) parseUnionType() *ast.UnionType {
	expr := &ast.UnionType{Token: p.curToken}
	p.nextToken()

	if !p.curTokenKindIs(lexer.TokenCurlyBraceOpen) {
		p.add(p.error(p.curToken, "expected curly brace open {, instead got ", p.curToken.Text))
		return nil
	}

	p.nextToken()

	if p.curTokenKindIs(lexer.TokenBracketClose) {
		p.nextToken()
		return &ast.UnionType{
			Token: expr.Token,
			Body:  []ast.Type{},
		}
	}

	for !p.curTokenKindIs(lexer.TokenCurlyBraceClose) {
		tp, err := p.parseType()

		if err != nil {
			p.add(p.error(p.curToken, err))
			p.sync(true)
		}

		expr.Body = append(expr.Body, tp)

		if !p.curTokenKindIs(lexer.TokenComma) {
			p.add(p.error(p.curToken, "expected a comma (,) at the end, instead got ", p.curToken.Text))
			p.syncUntilTokenIs(lexer.TokenCurlyBraceClose, true)
		}

		p.nextToken()
	}

	p.nextToken()

	return expr
}

func (p *Parser) parseSwitchExpression() ast.Expression {
	expr := &ast.SwitchExpression{Token: p.curToken}
	p.nextToken()

	// for switch
	if p.curTokenKindIs(lexer.TokenPartial) {
		expr.PartialCheck = true
		p.nextToken()
	}

	p.internalFlags = append(p.internalFlags, "if-mode")
	value := p.parseExpression(ASSIGN)
	p.internalFlags = slices.DeleteFunc(p.internalFlags, func(elem string) bool {
		return elem == "if-mode"
	})

	if value == nil {
		return nil
	}

	expr.Condition = value

	if !p.curTokenKindIs(lexer.TokenCurlyBraceOpen) {
		p.add(p.error(p.curToken, "expected curly brace open {, instead got ", p.curToken.Text))
		return nil
	}

	p.nextToken()

	cases, err := p.parseCases()

	if err != nil {
		p.add(err)
		return nil
	}

	expr.Cases = cases

	return expr
}

func (p *Parser) parseCases() ([]*ast.Case, error) {
	cases := make([]*ast.Case, 0)

round:
	for !p.curTokenKindIs(lexer.TokenCurlyBraceClose) {
		if !p.curTokenKindIs(lexer.TokenCase) {
			return cases, p.error(p.curToken, "expected case token, instead got ", p.curToken.Text)
		}

		cs := &ast.Case{Token: p.curToken, Break: true}
		p.nextToken()

		for !p.curTokenKindIs(lexer.TokenColon) {
			value := p.parseExpression(LOWEST)

			if value == nil {
				// ! This is a hacky way to support type in arm cases
				// try to parse the type in this case
				tp, err := p.parseType()
				if err != nil {
					p.syncUntilTokenIs(lexer.TokenCase, false)
					continue round
				}
				cs.ArmPattern = append(cs.ArmPattern, tp.(ast.Expression))
				// remove the last error in this case
				p.errors = p.errors[:len(p.errors)-1]
			}

			if value != nil {
				cs.ArmPattern = append(cs.ArmPattern, value)
			}

			if p.curTokenKindIs(lexer.TokenComma) && p.peekTokenKindIs(lexer.TokenColon) {
				return cases, p.error(p.curToken, "issue after last colo, probably u forgot to add another case, if no remove it")
			}

			if p.curTokenKindIs(lexer.TokenComma) {
				p.nextToken()
			}
		}

		p.nextToken()
		// here enter the body

		block := ast.BlockExpression{Token: p.curToken}
		block.Body = make([]ast.Statement, 0)

		for !p.curTokenKindIs(lexer.TokenCase) && !p.curTokenKindIs(lexer.TokenEOF) && !p.curTokenKindIs(lexer.TokenCurlyBraceClose) {
			if p.curTokenKindIs(lexer.TokenFallthrough) && p.peekTokenKindIs(lexer.TokenCase) {
				cs.Break = false
				p.nextToken()
			} else {
				// parse body expressions and statements
				stmt, err := p.parseStatement()

				if err != nil {
					p.add(err)
					p.sync(false)
				} else {
					block.Body = append(block.Body, stmt)
				}
			}
		}

		cs.Body = &block

		cases = append(cases, cs)
	}

	p.nextToken()

	return cases, nil
}

func (p *Parser) parseCastExpression() ast.Expression {
	expr := &ast.CastExpression{Token: p.curToken}
	p.nextToken()

	if !p.curTokenKindIs(lexer.TokenBraceOpen) {
		p.add(p.error(p.curToken, "expected brace open ) after cast keyword instead got ", p.curToken.Text))
		return nil
	}
	p.nextToken()

	if p.curTokenKindIs(lexer.TokenForce) {
		expr.ForceCast = true
		p.nextToken()
	}

	tp, err := p.parseType()

	if err != nil {
		p.add(err)
		return nil
	}

	expr.TargetType = tp

	if !p.curTokenKindIs(lexer.TokenBraceClose) {
		p.add(p.error(p.curToken, "expected brace close ) after ", tp, " instead got ", p.curToken.Text))
		return nil
	}

	p.nextToken()

	// ! Note: space currently isn't required, since it is better for the programmer readability and each one has his own style, currently don't force it
	expr.TargetExpression = p.parseExpression(LOWEST)

	return expr
}

// id, val in arr
func (p *Parser) parseIteratorIn() (*ast.IteratorIn, error) {
	expr := &ast.IteratorIn{Token: p.curToken}

	if !p.curTokenKindIs(lexer.TokenIdentifier) {
		return nil, p.error(p.curToken, "expected at least one identifier, instead got ", p.curToken.Text)
	}

	ident := p.parseIdentifier().(*ast.Identifier)

	expr.Identifiers = append(expr.Identifiers, ident)

	if p.curTokenKindIs(lexer.TokenComma) {
		p.nextToken()

		if !p.curTokenKindIs(lexer.TokenIdentifier) {
			return nil, p.error(p.curToken, "expected an identifier, instead got ", p.curToken.Text)
		}

		ident := p.parseIdentifier().(*ast.Identifier)

		expr.Identifiers = append(expr.Identifiers, ident)
	}

	if !p.curTokenKindIs(lexer.TokenIn) {
		return nil, p.error(p.curToken, "expected in keyword, instead got ", p.curToken.Text)
	}

	p.nextToken()

	target := p.parseExpression(OR)

	if p.curTokenKindIs(lexer.TokenRange) {
		pattern := &ast.RangePattern{Token: p.curToken, Start: target}
		p.nextToken()

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

		// patter expr
		expr.Target = pattern
	} else {
		expr.Target = target
	}

	return expr, nil
}

// let i = 0; i<10 ; i++
func (p *Parser) parseIterationPattern() (*ast.IterationPattern, error) {
	expr := &ast.IterationPattern{Token: p.curToken}

	start, err := p.parseDeclaration()

	if err != nil {
		return nil, err
	}

	expr.Start = start

	if !p.curTokenKindIs(lexer.TokenSemiColon) {
		return nil, p.error(p.curToken, "expected a ; after start expr, instead got ", p.curToken.Text)
	}

	p.nextToken()

	expr.Condition = p.parseExpression(LOWEST)

	if !p.curTokenKindIs(lexer.TokenSemiColon) {
		return nil, p.error(p.curToken, "expected a ; after condition expr, instead got ", p.curToken.Text)
	}

	p.nextToken()

	expr.End = p.parseExpression(OR)

	return expr, nil
}

func (p *Parser) parseForStatement() (*ast.ForStatement, error) {
	stmt := &ast.ForStatement{Token: p.curToken}
	p.nextToken()

	if p.curTokenKindIs(lexer.TokenLet) {
		// c-style loops
		pattern, err := p.parseIterationPattern()

		if err != nil {
			return nil, err
		}

		stmt.Pattern = pattern
	} else {
		switch p.peekToken.Kind {
		case lexer.TokenComma, lexer.TokenIn:
			// for in style loop

			pattern, err := p.parseIteratorIn()

			if err != nil {
				return nil, err
			}

			stmt.Pattern = pattern

		default:
			// for cond style loop
			expr := &ast.IterationPattern{Token: p.curToken}

			if !p.curTokenKindIs(lexer.TokenCurlyBraceOpen) && !p.curTokenKindIs(lexer.TokenDo) {
				expr.Condition = p.parseExpression(OR)
			}

			stmt.Pattern = expr
		}
	}

	if !p.curTokenKindIs(lexer.TokenCurlyBraceOpen) && !p.curTokenKindIs(lexer.TokenDo) {
		return nil, p.error(p.curToken, "expected curly brace open { or do token , instead got ", p.curToken.Text)
	}

	if p.curTokenKindIs(lexer.TokenCurlyBraceOpen) {
		p.nextToken()

		stmt.Body = p.parseBlockStatement().(*ast.BlockExpression)
	}

	if p.curTokenKindIs(lexer.TokenDo) {
		p.nextToken()
		s, err := p.parseStatement()

		if err != nil {
			p.add(err)
			p.sync(false)
		} else {
			stmt.Body = &ast.BlockExpression{
				Token: p.curToken,
				Body:  []ast.Statement{s},
			}
		}
	}

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

func (p *Parser) parseIdentifier() ast.Expression {
	ident := &ast.Identifier{Token: p.curToken, Value: p.curToken.Text}
	p.nextToken()
	return ident
}

func (p *Parser) parseDirectiveToValue() ast.Expression {
	var ident ast.Expression

	switch p.curToken.Kind {
	case lexer.TokenLine:
		ident = &ast.IntegerLiteral{Token: p.curToken, Value: int64(p.curToken.Row)}
	case lexer.TokenFile:
		ident = &ast.StringLiteral{Token: p.curToken, Value: p.filePath}
	case lexer.TokenDir:
		ident = &ast.StringLiteral{Token: p.curToken, Value: filepath.Dir(p.filePath)}
	}

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

	tp, err := p.parseType()

	if err != nil {
		p.add(err)
		return nil
	}

	expr.Type = tp

	if !p.curTokenKindIs(lexer.TokenCurlyBraceOpen) {
		p.add(p.error(expr.Token, "expected open curly brace {, instead got ", p.curToken.Text))
		return nil
	}

	// consume the [
	p.nextToken()

	elements := make([]ast.Expression, 0)

	if p.curTokenKindIs(lexer.TokenCurlyBraceClose) {
		p.nextToken()
		return expr
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
			continue round
		}

		elements = append(elements, val)
	}

	expr.Elements = elements

	if !p.curTokenKindIs(lexer.TokenCurlyBraceClose) {
		p.add(p.error(p.curToken, "expected close curly brace }, instead got ", p.curToken.Text))
		return nil
	}

	// consume ]
	p.nextToken()

	return expr
}

func (p *Parser) parseScope() (*ast.ScopeStatement, error) {
	stmt := &ast.ScopeStatement{Token: p.curToken}

	p.nextToken()

	stmt.Body = p.parseBlockStatement().(*ast.BlockExpression)
	return stmt, nil
}

func (p *Parser) parseMapLiteral() ast.Expression {
	prev := p.curToken

	tp, err := p.parseType()

	if err != nil {
		p.add(err)
		return nil
	}

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
			continue round
		}

		if !p.curTokenKindIs(lexer.TokenColon) {
			p.add(p.error(p.curToken, "expected colon : after key, instead got ", p.curToken.Text))
			p.sync(false)
			continue round
		}

		// consume :
		p.nextToken()

		value := p.parseExpression(LOWEST)

		if value == nil {
			p.sync(false)
			continue round
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
		Type:  tp,
		Pairs: pairs,
	}
}

// useful for parsing literal such as (integers of all sort, floats, strings, arrays, booleans ...etc)
func (p *Parser) parseLiteral() ast.Expression {
	switch p.curToken.Kind {
	case lexer.TokenInteger:
		return p.parseIntLiteral()
	case lexer.TokenFloat:
		return p.parseFloatLiteral()
	case lexer.TokenStr:
		return p.parseStringLiteral()
	case lexer.TokenChar:
		return p.parseCharLiteral()
	case lexer.TokenNul:
		return p.parseNulLiteral()
	case lexer.TokenBracketOpen:
		return p.parseArrayLiteral()
	case lexer.TokenMap:
		return p.parseMapLiteral()

	default:
		p.syncUntilTokenIs(lexer.TokenBraceClose, false)
		p.add(p.error(p.curToken, "expected a literal token (int, float, bool, ...etc), instead got ", p.curToken.Text))
		return nil
	}
}

func (p *Parser) parseCommentStatement() (*ast.Comment, error) {
	tok := p.curToken
	p.nextToken()
	return &ast.Comment{Token: tok, Value: tok.Text}, nil
}

func (p *Parser) parseGroupedExpression() ast.Expression {
	p.nextToken()
	exp := p.parseExpression(LOWEST)
	if !p.curTokenKindIs(lexer.TokenBraceClose) {
		p.add(p.error(p.curToken, "expected a brace close ), instead got ", p.curToken.Text))
		return nil
	}
	p.nextToken()
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
	if p.curTokenKindIs(lexer.TokenQuestion) || p.curTokenKindIs(lexer.TokenDo) {
		p.nextToken() // consume the ?

		exprStmt, err := p.parseExpressionStatement()
		if err != nil {
			return nil
		}
		expr.Consequence = &ast.BlockExpression{
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
		expr.Alternative = &ast.BlockExpression{
			Body: []ast.Statement{exprStmt},
		}
	} else {
		if !p.curTokenKindIs(lexer.TokenCurlyBraceOpen) {
			p.add(p.error(p.curToken, "expected close curly brace ( } ), instead got ", p.curToken.Text))
			return nil
		}

		p.nextToken()

		expr.Consequence = p.parseBlockStatement().(*ast.BlockExpression)

		// check if there is an else stmt
		if p.curTokenKindIs(lexer.TokenElse) {
			p.nextToken()
			// support for else if
			if p.curTokenKindIs(lexer.TokenIf) {
				expr.Alternative = p.parseIfExpression()
			} else {
				if !p.curTokenKindIs(lexer.TokenCurlyBraceOpen) {
					p.add(p.error(p.curToken, "expected curly brace open {, instead got ", p.curToken.Text))
					return nil
				}
				p.nextToken()

				expr.Alternative = p.parseBlockStatement()
			}
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

	args := p.parseArguments()
	expr.Args = args

	if !p.curTokenKindIs(lexer.TokenArrow) {
		p.add(p.error(p.curToken, "expected -> token after ), instead got ", p.curToken.Text))
		return nil
	}

	p.nextToken()

	if !p.curTokenKindIs(lexer.TokenBraceOpen) {
		p.add(p.error(p.curToken, "expected curly brace open (, instead got ", p.curToken.Text))
		return nil
	}

	p.nextToken()

	retTp := make([]ast.Type, 0)

	// check if ) is after (, means no arguments
	if p.curTokenKindIs(lexer.TokenBraceClose) {
		expr.Return.RtTypes = retTp
	} else {
		retTp, err := p.parseParamType()

		if err != nil {
			p.add(err)
			return nil
		}

		expr.Return.RtTypes = retTp
	}

	if !p.curTokenKindIs(lexer.TokenBraceClose) {
		p.add(p.error(p.curToken, "expected curly brace close ), instead got ", p.curToken.Text))
		return nil
	}

	p.nextToken()

	if p.curTokenKindIs(lexer.TokenMustUse) {
		expr.Return.MustUse = true
		p.nextToken()
	}

	if !p.curTokenKindIs(lexer.TokenCurlyBraceOpen) && !p.curTokenKindIs(lexer.TokenDo) {
		p.add(p.error(p.curToken, "expected curly brace open ( { ) or do token, instead got ", p.curToken.Text))
		return nil
	}

	if p.curTokenKindIs(lexer.TokenCurlyBraceOpen) {
		p.nextToken()

		body := p.parseBlockStatement().(*ast.BlockExpression)

		if body == nil {
			p.add(p.error(p.curToken, "expected valid body, instead got ", p.curToken.Text))
			return nil
		}

		expr.Body = body
	}

	if p.curTokenKindIs(lexer.TokenDo) {
		p.nextToken()
		s, err := p.parseStatement()

		if err != nil {
			p.add(err)
			p.sync(false)
		} else {
			expr.Body = &ast.BlockExpression{
				Token: p.curToken,
				Body:  []ast.Statement{s},
			}
		}
	}

	return expr
}

func (p *Parser) parseArguments() []*ast.Arg {
	// return another identifier which is
	args := make([]*ast.Arg, 0)

	if p.curTokenKindIs(lexer.TokenBraceClose) {
		p.nextToken()
		return args
	}

	arg := &ast.Arg{
		Token: p.curToken,
	}

	if !p.curTokenKindIs(lexer.TokenIdentifier) {
		p.add(p.error(p.curToken, "expected identifier, instead got ", p.curToken.Text))
		return nil
	}

	arg.Name = p.parseIdentifier().(*ast.Identifier)

	// expect colon
	if !p.curTokenKindIs(lexer.TokenColon) {
		p.add(p.error(p.curToken, "expected : after argument name, instead got ", p.curToken.Text))
		return nil
	}

	p.nextToken()

	// parse type
	tp, err := p.parseType()

	if err != nil {
		p.add(err)
		return nil
	}

	arg.Type = tp

	if p.curTokenKindIs(lexer.TokenAssign) {
		p.nextToken()
		// default value
		arg.DefaultValue = p.parseLiteral()
	}

	args = append(args, arg)

	for p.curTokenKindIs(lexer.TokenComma) {
		p.nextToken()

		arg := &ast.Arg{
			Token: p.curToken,
		}

		if !p.curTokenKindIs(lexer.TokenIdentifier) {
			p.add(p.error(p.curToken, "expected identifier, instead got ", p.curToken.Text))
			return nil
		}

		arg.Name = p.parseIdentifier().(*ast.Identifier)

		// expect colon
		if !p.curTokenKindIs(lexer.TokenColon) {
			p.add(p.error(p.curToken, "expected : after argument name, instead got ", p.curToken.Text))
			return nil
		}

		p.nextToken()

		// parse type
		tp, err := p.parseType()

		if err != nil {
			p.add(err)
			return nil
		}

		arg.Type = tp

		if p.curTokenKindIs(lexer.TokenAssign) {
			p.nextToken()
			// default value
			arg.DefaultValue = p.parseLiteral()
		}

		args = append(args, arg)
	}

	if !p.curTokenKindIs(lexer.TokenBraceClose) {
		p.add(p.error(p.curToken, "expected ) token in function definition, instead got ", p.curToken.Text))
		return nil
	}

	p.nextToken()

	return args
}

func (p *Parser) parseBlockStatement() ast.Expression {
	block := ast.BlockExpression{Token: p.curToken}
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
		p.add(p.error(p.curToken, "end of block expression expects }, instead got ", p.curToken.Text))
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

	exp := ast.CallExpression{Token: left.GetToken(), Function: left.(*ast.Identifier)}

	exp.Args = p.parseCallArguments()

	return &exp
}

func (p *Parser) parseCallArguments() []ast.Param {
	args := make([]ast.Param, 0)

	if !p.curTokenKindIs(lexer.TokenBraceOpen) {
		p.add(p.error(p.curToken, "expect brace (, instead got ", p.curToken.Text))
		return nil
	}

	p.nextToken()

	if p.curTokenKindIs(lexer.TokenBraceClose) {
		p.nextToken()
		return args
	}

	param := ast.Param{
		Token: p.curToken,
	}

	if p.curTokenKindIs(lexer.TokenIdentifier) {
		// parse it
		param.Name = p.parseIdentifier().(*ast.Identifier)

		if !p.curTokenKindIs(lexer.TokenAssign) {
			p.add(p.error(p.curToken, "expect assign = after ", param.Name, " instead got ", p.curToken.Text))
			return nil
		}
		p.nextToken()
	}

	param.Value = p.parseExpression(LOWEST)
	args = append(args, param)

	for p.curTokenKindIs(lexer.TokenComma) {
		p.nextToken()
		param := ast.Param{
			Token: p.curToken,
		}

		if p.curTokenKindIs(lexer.TokenIdentifier) {
			// parse it
			param.Name = p.parseIdentifier().(*ast.Identifier)

			if !p.curTokenKindIs(lexer.TokenAssign) {
				p.nextToken()
			}
		}

		param.Value = p.parseExpression(LOWEST)
		args = append(args, param)
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

	if p.curTokenKindIs(lexer.TokenRange) {
		goto rng
	}

	exp.Start = p.parseExpression(OR)

rng:
	if p.curTokenKindIs(lexer.TokenRange) {
		exp.Range = true
		p.nextToken()

		if !p.curTokenKindIs(lexer.TokenBracketClose) {
			exp.End = p.parseExpression(OR)
		}
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

func (p *Parser) parseNonExplicitStructInstanceExpression() ast.Expression {
	p.nextToken()

	if p.curTokenKindIs(lexer.TokenCurlyBraceOpen) {
		expr := &ast.StructInstanceExpression{Token: p.curToken}
		fields, err := p.parseFieldValues()

		if err != nil {
			p.add(err)
			return nil
		}

		expr.Body = fields
		return expr
	} else {
		if !p.curTokenKindIs(lexer.TokenIdentifier) {
			// error out
			p.error(p.curToken, "expected either a curly brace open {, or an identifier, instead got ", p.curToken.Text)
			p.sync(false)
			return nil
		}

		return p.parseIdentifier()
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
			Right:    p.parseExpression(OR),
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
				Token: lexer.Token{
					LiteralToken: lexer.LiteralToken{
						Kind: lexer.TokenInteger,
						Text: "1",
					},
					Row: expr.Left[0].GetToken().Row + 1,
					Col: expr.Left[0].GetToken().Row + 1,
				},
				Value: 1,
			},
		},
	}

	// consume the operator token (++, --)
	p.nextToken()

	return expr
}

func (p *Parser) parseMultiAssignStatement() (ast.Statement, error) {
	prev := p.curToken
	idents := []*ast.Identifier{p.parseIdentifier().(*ast.Identifier)}

	// collect lhs identifiers
	for p.curTokenKindIs(lexer.TokenComma) {
		p.nextToken() // eat comma
		if !p.curTokenKindIs(lexer.TokenIdentifier) {
			return nil, p.error(p.curToken, "expected identifier in multi-assign")
		}
		idents = append(idents, p.parseIdentifier().(*ast.Identifier))
	}

	// expect one of
	if !p.curTokenKindIs(lexer.TokenAssign) {
		return nil, p.error(p.curToken, "expected (:= , :: or =) operators, instead got ", p.curToken.Text)
	}

	p.nextToken() // move to first rhs expression

	exprs := []ast.Expression{p.parseExpression(LOWEST)}
	for p.curTokenKindIs(lexer.TokenComma) {
		p.nextToken() // eat comma
		exprs = append(exprs, p.parseExpression(LOWEST))
	}

	return &ast.AssignStatement{
		Token: prev,
		Left:  idents,
		Right: exprs,
	}, nil

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
