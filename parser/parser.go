package parser

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
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

// advances to the next token
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

func (p *Parser) Error(tok Token, msg string) error {
	space := fmt.Sprintf("%d   ", tok.Row)
	errMsg := fmt.Sprintf("\033[1;90m%v:%v:%v:\033[0m\n\n", p.FilePath, tok.Row, tok.Col)
	instruction := ""
	subSet := slices.DeleteFunc(p.Tokens, func(t Token) bool {
		return t.Row != tok.Row
	})
	for _, tk := range subSet {
		instruction += fmt.Sprintf(" %v", tk.Text)
	}
	errMsg += fmt.Sprintf("%s%s\n", space, instruction)
	for range len(space) + len(strings.Split(instruction, tok.Text)[0]) {
		errMsg += " "
	}
	errMsg += "\033[1;31m"
	for range len(tok.Text) {
		errMsg += "^"
	}
	errMsg += "\033[0m"
	errMsg += "\n" + msg
	return errors.New(errMsg)
}

func (p *Parser) Parse() *AST {

	ast := make(AST, 0)

	for p.Pos <= len(p.Tokens) {
		tok := p.peek()
		switch tok.Kind {
		case TokenPush, TokenConcat, TokenTrim, TokenExport, TokenSet, TokenThumbnailFrom, TokenUse, TokenProcess, TokenIf, TokenElse, TokenForEach, TokenSkip:
			node, err := p.parseCommand()
			if err != nil {
				fmt.Println(err)
				return nil
			}
			ast = append(ast, *node)

		case TokenEOF:
			return &ast
		case TokenError:
			fmt.Println(tok.Text)
			return nil
		default:
			if tok.Kind == TokenCurlyBraceClose || tok.Kind == TokenCurlyBraceOpen {
				fmt.Printf("unexpected brace token outside of a command process at line %d\n", tok.Row)
				return nil
			}
			fmt.Printf("unexpected token %s at line %d col %v\n", tok.Text, tok.Row, tok.Col)
			return nil
		}
	}

	return &ast
}

func (p *Parser) parseCommand() (*StatementNode, error) {
	cmdToken := p.next() // Consume command

	position := Position{
		Col: cmdToken.Col,
		Row: cmdToken.Row,
	}

	// Validation step
	switch cmdToken.Kind {
	case TokenPush:
		return p.pushHandler(position)
	case TokenTrim:
		return p.trimHandler(position)
	case TokenConcat:
		return p.concatHandler(position)
	case TokenThumbnailFrom:
		return p.thumbnailHandler(position)
	case TokenExport:
		return p.exportHandler(position)
	case TokenSet:
		return p.setHandler(position)
	case TokenUse:
		return p.useHandler(position)
	case TokenProcess:
		return p.processHandler(position)
	case TokenIf:
		return p.ifHandler(position)
	case TokenElse:
		return p.elseHandler(position)
	case TokenForEach:
		return p.foreachHandler(position)
	case TokenSkip:
		return p.skipHandler(position)
	}

	// All good, create AST node
	return &StatementNode{}, fmt.Errorf("ERROR: unexpected token appeared, line %v row%v", cmdToken.Row, cmdToken.Col)
}

func (p *Parser) parseMemberAccess() (*MemberAccessExpression, error) {
	tok, err := p.expect([]TokenKind{TokenIdentifier})

	if err != nil {
		errMsg := fmt.Sprintf("ERROR: expected %v, got %v", TokenIdentifier, tok.Kind)
		return nil, p.Error(tok, errMsg)
	}

	identifierName := tok.Text

	tok = p.next()

	if tok.Kind == TokenDot {
		property, err := p.parseMemberAccess()

		if err != nil {
			return nil, err
		}

		return &MemberAccessExpression{
			Name:     identifierName,
			Property: property,
		}, nil
	} else {
		p.Pos--
		return &MemberAccessExpression{
			Name: identifierName,
		}, nil
	}
}

func (p *Parser) foreachHandler(pos Position) (*StatementNode, error) {
	args := make([]any, 0)
	tok, err := p.expect([]TokenKind{TokenIdentifier})

	if err != nil {
		errMsg := fmt.Sprintf("ERROR: in foreach, a variable name is expected, got %v", tok.Kind)
		return nil, p.Error(tok, errMsg)
	}

	args = append(args, ExpressionNode{
		Type:     IdentifierExpression,
		Value:    tok.Text,
		ExprType: IdentifierType,
		Position: Position{
			Col: tok.Col,
			Row: tok.Row,
		},
	})

	_, err = p.expect([]TokenKind{TokenIn})

	if err != nil {
		errMsg := fmt.Sprintf("ERROR: in foreach after key name: %v, %v is expected", args[len(args)-1].(ExpressionNode).Value, TokenIn)
		return nil, p.Error(tok, errMsg)

	}

	tok, err = p.expect([]TokenKind{TokenString, TokenIdentifier})

	if err != nil {
		errMsg := "ERROR: in foreach after in keyword, either a string or reference to a string"
		return nil, p.Error(tok, errMsg)

	}

	if tok.Kind == TokenIdentifier {
		args = append(args, ExpressionNode{
			Type:     IdentifierExpression,
			Value:    tok.Text,
			ExprType: IdentifierType,
			Position: Position{
				Col: tok.Col,
				Row: tok.Row,
			},
		})
	}

	if tok.Kind == TokenString {
		args = append(args, ExpressionNode{
			Type:     LiteralExpression,
			Value:    tok.Text,
			ExprType: StringType,
			Position: Position{
				Col: tok.Col,
				Row: tok.Row,
			},
		})
	}

	tok = p.next()

	if tok.Kind == TokenRecurse {
		expression := ExpressionNode{
			Type:       IdentifierExpression,
			Identifier: tok.Text,
			// recurse until the last folder
			Value:    "infinity",
			ExprType: IdentifierType,
			Position: Position{
				Col: tok.Col,
				Row: tok.Row,
			},
		}

		tok = p.next()
		if tok.Kind == TokenNumber {
			num, err := strconv.Atoi(tok.Text)
			if err != nil {
				return nil, err
			}
			expression.Value = num
		}
		args = append(args, expression)
	} else {
		p.Pos--
	}

	tok, err = p.expect([]TokenKind{TokenCurlyBraceOpen})

	if err != nil {
		errMsg := ""
		lastParam := args[len(args)-1].(ExpressionNode)
		if lastParam.Identifier == "recurse" {
			errMsg = fmt.Sprintf("ERROR: in foreach, after recurse keyword, we expect Open Curly Brace ( { ), not %v", tok.Text)
		} else if lastParam.ExprType == IdentifierType {
			errMsg = fmt.Sprintf("ERROR: in foreach, after the defined %v reference, we expect Open Curly Brace ( { ), not %v", lastParam.Value, tok.Text)
		} else {
			// This means that this is a string
			errMsg = fmt.Sprintf(`ERROR: in foreach, after "%v" string, we expect Open Curly Brace ( { ), not %v`, lastParam.Value, tok.Text)
		}
		return nil, p.Error(tok, errMsg)
	}
	body := make(AST, 0)

	for p.peek().Kind != TokenCurlyBraceClose {
		ast, err := p.parseCommand()
		if err != nil {
			return nil, err
		}
		body = append(body, *ast)
	}

	tok, err = p.expect([]TokenKind{TokenCurlyBraceClose})

	if err != nil {
		errMsg := fmt.Sprintf("ERROR: in the end of the foreach statement, a Close Curly Brace ( } ) is expected, not %v", tok.Text)
		return nil, p.Error(tok, errMsg)
	}

	return &StatementNode{
		Type:     ForeachStatement,
		Params:   args,
		Body:     body,
		Position: pos,
		Order:    p.Pos,
	}, nil
}

func (p *Parser) skipHandler(pos Position) (*StatementNode, error) {
	tok := p.peek()

	if tokenKey, isMatched := keywords[tok.Text]; isMatched {
		if tokenKey == TokenBool {
			errMsg := "ERROR: skip statement, doesn't expect any additional tokens"
			return nil, p.Error(tok, errMsg)
		}
	}
	return &StatementNode{
		Type:     SkipStatement,
		Params:   []any{},
		Position: pos,
		Order:    p.Pos,
	}, nil
}

func (p *Parser) ifHandler(pos Position) (*StatementNode, error) {
	args := make([]any, 0)

	tok, err := p.expect([]TokenKind{TokenIdentifier, TokenExclamation})

	if err != nil {
		errMsg := "ERROR: if expects, either a string or reference to a string"
		return nil, p.Error(tok, errMsg)
	}
	if tok.Kind == TokenIdentifier {
		p.Pos--

		prop, err := p.parseMemberAccess()

		if err != nil {
			return nil, err
		}

		leftIdExpr := ExpressionNode{
			Type:     IdentifierExpression,
			Value:    prop,
			ExprType: IdentifierType,
			Position: Position{
				Row: tok.Row,
				Col: tok.Col,
			},
		}

		tok = p.next()

		switch tok.Kind {
		case TokenEquals, TokenGreater, TokenGreaterOrEqual, TokenLess, TokenLessOrEqual:
			binExpr := BinaryExpressionNode{
				Type:     BinaryExpression,
				Left:     leftIdExpr,
				Operator: binOperators[tok.Kind],
			}

			tok = p.next()
			switch tok.Kind {
			case TokenString:
				binExpr.Right = ExpressionNode{
					Type:     LiteralExpression,
					Value:    tok.Text,
					ExprType: StringType,
					Position: Position{
						Col: tok.Col,
						Row: tok.Row,
					},
				}
			case TokenNumber:
				num, err := strconv.ParseFloat(tok.Text, 64)
				if err != nil {
					errMsg := fmt.Sprintf("invalid number format, %v", err)
					return nil, p.Error(tok, errMsg)
				}
				binExpr.Right = ExpressionNode{
					Type:     LiteralExpression,
					Value:    num,
					ExprType: NumberType,
					Position: Position{
						Col: tok.Col,
						Row: tok.Row,
					},
				}
			case TokenMinus:
				// for negative numbers
				tok := p.next()
				num, err := strconv.ParseFloat(tok.Text, 64)
				if err != nil {
					errMsg := fmt.Sprintf("invalid number format, %v", err)
					return nil, p.Error(tok, errMsg)
				}
				binExpr.Right = ExpressionNode{
					Type:     LiteralExpression,
					Value:    -num,
					ExprType: NumberType,
					Position: Position{
						Col: tok.Col,
						Row: tok.Row,
					},
				}
			case TokenIdentifier:
				binExpr.Right = ExpressionNode{
					Type:     IdentifierExpression,
					Value:    tok.Text,
					ExprType: IdentifierType,
					Position: Position{
						Col: tok.Col,
						Row: tok.Row,
					},
				}
			default:
				errMsg := fmt.Sprintf("ERROR: the right side of an if statement expects one of this option types (string|number|defined variable), got %v", tok.Text)
				return nil, p.Error(tok, errMsg)
			}
			args = append(args, binExpr)
		case TokenCurlyBraceOpen:
			p.Pos--
			args = append(args, leftIdExpr)
		default:
			errMsg := fmt.Sprintf("ERROR: a binary expression expects one of this options ( == | <= | >= | < | >), got %v", tok.Kind)
			return nil, p.Error(tok, errMsg)
		}
	}

	if tok.Kind == TokenExclamation {
		// for the !
		unaryExpr := UnaryExpressionNode{
			Type:     UnaryExpression,
			Operator: unaryOperators[tok.Text],
		}

		tok, err = p.expect([]TokenKind{TokenIdentifier})
		if err != nil {
			errMsg := fmt.Sprintf("ERROR: right of side of expression, expects boolean type value, got %v", tok.Kind)
			return nil, p.Error(tok, errMsg)
		}
		p.Pos--

		prop, err := p.parseMemberAccess()

		if err != nil {
			return nil, err
		}

		unaryExpr.Right = ExpressionNode{
			Type:     IdentifierExpression,
			Value:    prop,
			ExprType: IdentifierType,
			Position: Position{
				Row: tok.Row,
				Col: tok.Col,
			},
		}
	}

	tok, err = p.expect([]TokenKind{TokenCurlyBraceOpen})
	if err != nil {
		errMsg := fmt.Sprintf("ERROR: after %v value, expect a Curly Brace Open ( { ), got %v", args[len(args)-1].(ExpressionNode).Value, tok.Text)
		return nil, p.Error(tok, errMsg)
	}
	body := make(AST, 0)

	for p.peek().Kind != TokenCurlyBraceClose {
		ast, err := p.parseCommand()
		if err != nil {
			return nil, err
		}
		body = append(body, *ast)
	}

	tok, err = p.expect([]TokenKind{TokenCurlyBraceClose})

	if err != nil {
		errMsg := fmt.Sprintf("ERROR: in the end of the if statement, a Close Curly Brace ( } ) is expected, not %v", tok.Text)
		return nil, p.Error(tok, errMsg)
	}

	return &StatementNode{
		Type:     IfStatement,
		Params:   args,
		Body:     body,
		Position: pos,
		Order:    p.Pos,
	}, nil
}

func (p *Parser) elseHandler(pos Position) (*StatementNode, error) {
	tok := p.next()

	switch tok.Kind {
	case TokenIf:
		return p.ifHandler(pos)
	case TokenCurlyBraceOpen:
		body := make(AST, 0)

		for p.peek().Kind != TokenCurlyBraceClose {
			ast, err := p.parseCommand()
			if err != nil {
				return nil, err
			}
			body = append(body, *ast)
		}

		tok, err := p.expect([]TokenKind{TokenCurlyBraceClose})

		if err != nil {
			errMsg := fmt.Sprintf("ERROR: in the end of the else statement, a Close Curly Brace ( } ) is expected, not %v", tok.Text)
			return nil, p.Error(tok, errMsg)
		}

		return &StatementNode{
			Type:     ElseStatement,
			Params:   []any{},
			Body:     body,
			Position: pos,
			Order:    p.Pos,
		}, nil
	default:
		errMsg := fmt.Sprintf("ERROR: else statement, expects either if statement or nothing, got %v", tok.Kind)
		return nil, p.Error(tok, errMsg)
	}
}

func (p *Parser) pushHandler(pos Position) (*StatementNode, error) {

	args := make([]any, 0)

	tok, err := p.expect([]TokenKind{TokenIdentifier, TokenString})
	if err != nil {
		errMsg := fmt.Sprintf("ERROR: push statement, expects either string or string reference, got %v", tok.Kind)
		return nil, p.Error(tok, errMsg)
	}
	if tok.Kind == TokenString {
		// check the param format
		// the param format needs to be a valid path
		args = append(args, ExpressionNode{
			Type:     LiteralExpression,
			Value:    tok.Text,
			ExprType: StringType,
			Position: Position{
				Col: tok.Col,
				Row: tok.Row,
			},
		})

	}

	if tok.Kind == TokenIdentifier {
		args = append(args, ExpressionNode{
			Type:     IdentifierExpression,
			Value:    tok.Text,
			ExprType: IdentifierType,
			Position: Position{
				Col: tok.Col,
				Row: tok.Row,
			},
		})
	}

	return &StatementNode{
		Type:     PushStatement,
		Params:   args,
		Position: pos,
		Order:    p.Pos,
	}, nil
}

func (p *Parser) trimHandler(pos Position) (*StatementNode, error) {
	args := make([]any, 0)

	for index := range 2 {
		tok, err := p.expect([]TokenKind{TokenTime})
		if err != nil {
			errMsg := ""
			if index == 1 {
				errMsg = fmt.Sprintf("ERROR: trim statement, expects start to be of time type, got %v", tok.Kind)
			}
			if index == 2 {
				errMsg = fmt.Sprintf("ERROR: trim statement, expects start to be of time type, got %v", tok.Kind)
			}
			return nil, p.Error(tok, errMsg)
		}
		args = append(args, ExpressionNode{
			Type:     LiteralExpression,
			Value:    tok.Text,
			ExprType: TimeType,
			Position: Position{
				Row: tok.Row,
				Col: tok.Col,
			},
		})
	}

	// check the format of the path if it exists
	tok := p.next()

	switch tok.Kind {
	case TokenString:
		args = append(args, ExpressionNode{
			Type:     LiteralExpression,
			Value:    tok.Text,
			ExprType: StringType,
			Position: Position{
				Row: tok.Row,
				Col: tok.Col,
			},
		})

	case TokenIdentifier:
		args = append(args, ExpressionNode{
			Type:     LiteralExpression,
			Value:    tok.Text,
			ExprType: IdentifierType,
			Position: Position{
				Row: tok.Row,
				Col: tok.Col,
			},
		})
	default:
		p.Pos--
		args = append(args, ExpressionNode{
			Type:     LiteralExpression,
			ExprType: StringType,
			Value:    "last", // indicates that this will get applied at the last element in the stack
			Position: Position{
				Row: tok.Row,
				Col: tok.Col,
			},
		})
	}

	return &StatementNode{
		Type:     TrimStatement,
		Params:   args,
		Position: pos,
		Order:    p.Pos,
	}, nil
}

func (p *Parser) concatHandler(pos Position) (*StatementNode, error) {
	tok := p.peek()

	if tokenKey, isMatched := keywords[tok.Text]; isMatched {
		if tokenKey == TokenBool {
			errMsg := "ERROR: concat statement, doesn't expect any additional tokens"
			return nil, p.Error(tok, errMsg)
		}
	}
	return &StatementNode{
		Type:     ConcatStatement,
		Params:   []any{},
		Position: pos,
		Order:    p.Pos,
	}, nil
}

func (p *Parser) thumbnailHandler(pos Position) (*StatementNode, error) {
	args := make([]any, 0)

	tok := p.next()

	format := tok.Text

	switch tok.Kind {
	case TokenTime:
		timeFormat := `^\d{2}:\d{2}:\d{2}$`
		format := tok.Text
		if matched, _ := regexp.MatchString(timeFormat, format); matched {

			args = append(args, ExpressionNode{
				Type:     LiteralExpression,
				Value:    tok.Text,
				ExprType: TimeType,
				Position: Position{
					Row: tok.Row,
					Col: tok.Col,
				},
			})
		}
	case TokenNumber:
		num, err := strconv.Atoi(format)
		if err != nil {
			errMsg := fmt.Sprintf("invalid number format, %v", err)
			return nil, p.Error(tok, errMsg)
		}
		args = append(args, ExpressionNode{
			Type:     LiteralExpression,
			Value:    num,
			ExprType: NumberType,
			Position: Position{
				Row: tok.Row,
				Col: tok.Col,
			},
		})
	default:
		errMsg := fmt.Sprintf("ERROR: thumbnail from first args is either of type number or time, got %v", tok.Kind)
		return nil, p.Error(tok, errMsg)
	}

	tok, err := p.expect([]TokenKind{TokenString, TokenIdentifier})
	if err != nil {
		errMsg := fmt.Sprintf("ERROR: thumbnail from second args is either of type string or string reference, got %v", tok.Kind)
		return nil, p.Error(tok, errMsg)
	}
	// this may return an error cause it forces to use a video format only
	if tok.Kind == TokenString {
		args = append(args, ExpressionNode{
			Type:     LiteralExpression,
			Value:    tok.Text,
			ExprType: StringType,
			Position: Position{
				Row: tok.Row,
				Col: tok.Col,
			},
		})
	}

	if tok.Kind == TokenIdentifier {
		args = append(args, ExpressionNode{
			Type:     IdentifierExpression,
			Value:    tok.Text,
			ExprType: IdentifierType,
			Position: Position{
				Row: tok.Row,
				Col: tok.Col,
			},
		})
	}

	return &StatementNode{
		Type:     ThumbnailStatement,
		Params:   args,
		Position: pos,
		Order:    p.Pos,
	}, nil
}

func (p *Parser) processHandler(pos Position) (*StatementNode, error) {
	args := make([]any, 0)
	tok, err := p.expect([]TokenKind{TokenIdentifier})
	if err != nil {
		errMsg := fmt.Sprintf("ERROR: process block expects, an name, got %v", tok.Kind)
		return nil, p.Error(tok, errMsg)
	}
	args = append(args, ExpressionNode{
		Type:     IdentifierExpression,
		Value:    tok.Text,
		ExprType: IdentifierType,
		Position: Position{
			Row: tok.Row,
			Col: tok.Col,
		},
	})

	tok, err = p.expect([]TokenKind{TokenCurlyBraceOpen})
	if err != nil {
		errMsg := fmt.Sprintf("ERROR: after %v value, expect a Curly Brace Open ( { ), got %v", args[len(args)-1].(ExpressionNode).Value, tok.Text)
		return nil, p.Error(tok, errMsg)
	}

	body := make(AST, 0)

	for p.peek().Kind != TokenCurlyBraceClose {
		ast, err := p.parseCommand()

		if err != nil {
			return nil, err
		}

		// this part, doesn't allow for nested blocks
		if ast.Type != ProcessStatement {
			body = append(body, *ast)
		} else {
			return nil, fmt.Errorf("ERROR: process isn't allowed inside of another process")
		}

	}

	tok, err = p.expect([]TokenKind{TokenCurlyBraceClose})

	if err != nil {
		errMsg := fmt.Sprintf("ERROR: at the end process block, expect a Curly Brace Close ( } ), got %v", tok.Text)
		return nil, p.Error(tok, errMsg)
	}

	return &StatementNode{
		Type:     ProcessStatement,
		Params:   args,
		Body:     body,
		Position: pos,
		Order:    p.Pos,
	}, nil
}

func (p *Parser) setHandler(pos Position) (*StatementNode, error) {
	args := make([]any, 0)
	tok, err := p.expect([]TokenKind{TokenIdentifier})
	if err != nil {
		errMsg := fmt.Sprintf("ERROR: process block expects, an name, got %v", tok.Kind)
		return nil, p.Error(tok, errMsg)
	}
	expression := ExpressionNode{
		Type:       IdentifierExpression,
		Identifier: tok.Text,
		ExprType:   IdentifierType,
		Position: Position{
			Row: tok.Row,
			Col: tok.Col,
		},
	}

	tok = p.next()

	objPos := Position{}

	var value any
	// different types of definition
	switch tok.Kind {
	case TokenString:
		value = tok.Text
	case TokenBool:
		value = tok.Text == "true"
	case TokenMinus:
		tok = p.next()
		// parse the value first then append it to the args array
		num, err := strconv.ParseFloat(tok.Text, 64)
		if err != nil {
			return nil, err
		}

		value = -num

	case TokenPlus:
		tok = p.next()
		// parse the value first then append it to the args array
		num, err := strconv.ParseFloat(tok.Text, 64)
		if err != nil {
			errMsg := "invalid number, format"
			return nil, p.Error(tok, errMsg)
		}

		value = num

	case TokenNumber:
		num, err := strconv.ParseFloat(tok.Text, 64)
		if err != nil {
			errMsg := "invalid number, format"
			return nil, p.Error(tok, errMsg)
		}

		value = num

	case TokenIdentifier:
		p.Pos--
		prop, err := p.parseMemberAccess()

		if err != nil {
			return nil, err
		}

		value = prop

	case TokenCurlyBraceOpen:

		objValue := make(ObjectLiteral)

		objPos.Col = tok.Col
		objPos.Row = tok.Row

		for p.peek().Kind != TokenCurlyBraceClose {
			key, err := p.expect([]TokenKind{TokenIdentifier})

			if err != nil {
				errMsg := fmt.Sprintf("ERROR: in object body keyname is expected, got %s", key.Kind)
				return nil, p.Error(key, errMsg)
			}

			objValue[key.Text] = ExpressionNode{}

			colon, err := p.expect([]TokenKind{TokenColon})

			if err != nil {
				errMsg := fmt.Sprintf("ERROR: after keyname (%v) a colon (:) is expected, got %s", key.Text, colon.Kind)
				return nil, p.Error(key, errMsg)
			}

			nextTok := p.next()

			var val ExpressionNode
			switch nextTok.Kind {
			case TokenString:
				val = ExpressionNode{
					Type:     LiteralExpression,
					Value:    nextTok.Text,
					ExprType: StringType,
					Position: Position{
						Row: key.Row,
						Col: key.Col,
					},
				}

			case TokenNumber:
				num, err := strconv.ParseFloat(nextTok.Text, 64)
				if err != nil {
					errMsg := "invalid number format"
					return nil, p.Error(nextTok, errMsg)
				}
				val = ExpressionNode{
					Type:     LiteralExpression,
					Value:    num,
					ExprType: NumberType,
					Position: Position{
						Row: key.Row,
						Col: key.Col,
					},
				}
			case TokenBool:
				val = ExpressionNode{
					Type:     LiteralExpression,
					Value:    nextTok.Text == "true",
					ExprType: BooleanType,
					Position: Position{
						Row: key.Row,
						Col: key.Col,
					},
				}
			default:
				errMsg := fmt.Sprintf("ERROR: unsupported type %v", nextTok.Text)
				return nil, p.Error(nextTok, errMsg)
			}

			objValue[key.Text] = val
		}
		value = objValue

		p.expect([]TokenKind{TokenCurlyBraceClose})

	default:
		errMsg := fmt.Sprintf("ERROR, object body expects a key-value pair, got %v", tok.Kind)

		return nil, p.Error(tok, errMsg)
	}

	expression.Value = value
	args = append(args, expression)

	return &StatementNode{
		Type:     SetStatement,
		Params:   args,
		Position: pos,
		Order:    p.Pos,
	}, nil
}

func (p *Parser) useHandler(pos Position) (*StatementNode, error) {
	args := make([]any, 0)
	tok, err := p.expect([]TokenKind{TokenIdentifier})
	if err != nil {
		errMsg := fmt.Sprintf("ERROR: use statement, expects a variable reference, got %v", tok.Kind)
		return nil, p.Error(tok, errMsg)
	}
	args = append(args, ExpressionNode{
		Type:     IdentifierExpression,
		Value:    tok.Text,
		ExprType: IdentifierType,
		Position: Position{
			Row: tok.Row,
			Col: tok.Col,
		},
	})

	tok = p.next()

	switch tok.Kind {
	case TokenOn:
		tok, err := p.expect([]TokenKind{TokenIdentifier})

		if err != nil {
			errMsg := fmt.Sprintf("ERROR: use statement, after %v expects on keyword, got %v", args[len(args)-1].(ExpressionNode).Value, tok.Text)
			return nil, p.Error(tok, errMsg)
		}

		if tok.Kind == TokenString {
			args = append(args, ExpressionNode{
				Type:     LiteralExpression,
				Value:    tok.Text,
				ExprType: StringType,
				Position: Position{
					Row: tok.Row,
					Col: tok.Col,
				},
			})

		} else {
			args = append(args, ExpressionNode{
				Type:     IdentifierExpression,
				Value:    tok.Text,
				ExprType: IdentifierType,
				Position: Position{
					Row: tok.Row,
					Col: tok.Col,
				},
			})
		}

	default:
		p.Pos--
	}

	return &StatementNode{
		Type:     UseStatement,
		Params:   args,
		Position: pos,
		Order:    p.Pos,
	}, nil
}

func (p *Parser) exportHandler(pos Position) (*StatementNode, error) {
	args := make([]any, 0)

	tok, err := p.expect([]TokenKind{TokenIdentifier, TokenString})
	if err != nil {
		errMsg := fmt.Sprintf("ERROR: push statement, expects either string or string reference, got %v", tok.Kind)
		return nil, p.Error(tok, errMsg)
	}
	// check the param format
	if tok.Kind == TokenString {
		args = append(args, ExpressionNode{
			Type:     LiteralExpression,
			Value:    tok.Text,
			ExprType: StringType,
			Position: Position{
				Row: tok.Row,
				Col: tok.Col,
			},
		})
	}

	if tok.Kind == TokenIdentifier {
		args = append(args, ExpressionNode{
			Type:     IdentifierExpression,
			Value:    tok.Text,
			ExprType: IdentifierType,
			Position: Position{
				Row: tok.Row,
				Col: tok.Col,
			},
		})
	}

	return &StatementNode{
		Type:     ExportStatement,
		Params:   args,
		Position: pos,
		Order:    p.Pos,
	}, nil
}
