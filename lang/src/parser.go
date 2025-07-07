package src

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type Command = string

const (
	PUSH           Command = "push"
	TRIM           Command = "trim"
	EXPORT         Command = "export"
	CONCAT         Command = "concat"
	THUMBNAIL_FROM Command = "thumbnail_from"
	SET            Command = "set"
	USE            Command = "use"
	BLOCK          Command = "block"
)

type (
	Statement  = string
	Expression = string
	Type       = string
)

const (
	// Statements
	PushStatement      Statement = "push"
	TrimStatement      Statement = "trim"
	ExportStatement    Statement = "export"
	ConcatStatement    Statement = "concat"
	ThumbnailStatement Statement = "thumbnail_from"
	SetStatement       Statement = "set"
	UseStatement       Statement = "use"
	BlockStatement     Statement = "block"

	// Expression
	LiteralExpression    Expression = "literal_expression"
	IdentifierExpression Expression = "identifier_expression"
	ObjectExpression     Expression = "object_expression"

	// Types
	// Primitives
	NumberType  Type = "number"
	BooleanType Type = "bool"
	StringType  Type = "filepath"
	// Custom
	IdentifierType Type = "identifier"
	TimeType       Type = "time"
	// Complex
	ObjectType Type = "object"
)

var (
	videoExts = []string{
		".mp4", ".mov", ".avi", ".mkv",
		".webm", ".flv", ".wmv",
	}
	imageExts = []string{
		".jpg", ".jpeg", ".png", ".gif",
		".bmp", ".webp", ".tiff",
	}
)

type Position struct {
	Row int
	Col int
}

type ExpressionNode struct {
	Type     Expression // "literal_expression", "identifier_expression", etc.
	Value    any        // string, float64, bool, or even ObjectLiteral
	ExprType Type       // For type-checking: "number", "bool", etc.
	Position Position
}

type StatementNode struct {
	Type     Statement // e.g., "push", "set", etc.
	Params   []ExpressionNode
	Body     []StatementNode // Only for block/batch/etc.
	Position Position
	Order    int
}

// we use this when an expression is an object expression
type ObjectLiteral map[string]ExpressionNode

type AST = []StatementNode

type Parser struct {
	Tokens []Token
	Pos    int
}

func NewParser(tokens []Token) *Parser {
	return &Parser{
		Tokens: tokens,
		Pos:    0,
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

func (p *Parser) Parse() *AST {

	ast := make(AST, 0)

	for p.Pos <= len(p.Tokens) {
		tok := p.peek()

		switch tok.Kind {
		case TokenPush, TokenConcat, TokenTrim, TokenExport, TokenSet, TokenThumbnailFrom, TokenUse, TokenBlock:
			node, err := p.parseCommand()
			if err != nil {
				fmt.Println(err)
				return nil
			}
			ast = append(ast, *node)

		case TokenEOF:
			return &ast

		default:
			if tok.Kind == TokenCurlyBraceClose || tok.Kind == TokenCurlyBraceOpen {
				fmt.Printf("unexpected brace token outside of a command block at line %d\n", tok.Row)
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
	case TokenBlock:
		return p.blockHandler(position)
	}

	// All good, create AST node
	return &StatementNode{}, fmt.Errorf("ERROR: unexpected token appeared, line %v row%v", cmdToken.Row, cmdToken.Col)
}

func (p *Parser) videoPathCheck(path string) error {
	path = strings.TrimSpace(path)

	_, err := isValidPathFormat(path)
	if err != nil {
		return err
	}

	osPath, _ := os.Getwd()
	path = filepath.Join(osPath, path)
	path = filepath.Clean(path)

	if !checkFileIsOfTypeMode(path, VIDEO) {
		return errors.New("ERROR: file extension needs to be a video")
	}

	return nil
}

func (p *Parser) imagePathCheck(path string) error {
	path = strings.TrimSpace(path)

	_, err := isValidPathFormat(path)
	if err != nil {
		return err
	}

	osPath, _ := os.Getwd()
	path = filepath.Join(osPath, path)
	path = filepath.Clean(path)

	if !checkFileIsOfTypeMode(path, IMAGE) {
		return errors.New("ERROR: file extension needs to be a image")
	}

	return nil
}

func (p *Parser) pushHandler(pos Position) (*StatementNode, error) {
	args := make([]ExpressionNode, 0)

	tok := p.next()

	if tok.Kind != TokenString && tok.Kind != TokenIdentifier {
		return nil, fmt.Errorf("ERROR: unexpected value at line %v, row %v\npush command takes only string param", tok.Row, tok.Col)
	}

	if tok.Kind == TokenString {
		path := tok.Text
		// check the param format
		// the param format needs to be a valid path
		if err := p.videoPathCheck(path); err != nil {
			return nil, err
		}
		args = append(args, ExpressionNode{
			Type:     LiteralExpression,
			ExprType: StringType,
			Value:    tok.Text,
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
	args := make([]ExpressionNode, 0)

	for range 2 {
		tok := p.next()
		if tok.Kind != TokenTime {
			return nil, fmt.Errorf("ERROR: expected %v, got %v", TokenTime, tok.Kind)
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
		if err := p.videoPathCheck(tok.Text); err != nil {
			return nil, err
		}

		args = append(args, ExpressionNode{
			Type:     LiteralExpression,
			ExprType: StringType,
			Value:    tok.Text,
			Position: Position{
				Row: tok.Row,
				Col: tok.Col,
			},
		})

	case TokenIdentifier:
		args = append(args, ExpressionNode{
			Type:     LiteralExpression,
			ExprType: StringType,
			Value:    tok.Text,
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
			return nil, fmt.Errorf("ERROR: expected nothing, got %v", TokenBool)
		}
	}
	return &StatementNode{
		Type:     ConcatStatement,
		Params:   []ExpressionNode{},
		Position: pos,
		Order:    p.Pos,
	}, nil
}

func (p *Parser) thumbnailHandler(pos Position) (*StatementNode, error) {
	args := make([]ExpressionNode, 0)

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
			return nil, fmt.Errorf("invalid number format, %v", err)
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
		return nil, fmt.Errorf("ERROR: expected (%v, %v), got %v", TokenTime, TokenNumber, tok.Kind)
	}

	tok = p.next()

	if tok.Kind != TokenString && tok.Kind != TokenIdentifier {
		return nil, fmt.Errorf("ERROR: expected %v, got %v", TokenString, tok.Kind)
	}

	// this may return an error cause it forces to use a video format only
	if tok.Kind == TokenString {
		if err := p.imagePathCheck(tok.Text); err != nil {
			return nil, err
		}

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

func (p *Parser) blockHandler(pos Position) (*StatementNode, error) {
	args := make([]ExpressionNode, 0)
	tok := p.next()

	if tok.Kind != TokenIdentifier {
		return nil, fmt.Errorf("ERROR: expect a %v, got %v", TokenIdentifier, tok.Kind)
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

	if tok.Kind != TokenCurlyBraceOpen {
		return nil, fmt.Errorf("ERROR: expect a %v, got %v", TokenCurlyBraceOpen, tok.Kind)
	}

	body := make(AST, 0)

	for p.peek().Kind != TokenCurlyBraceClose {
		ast, err := p.parseCommand()

		if err != nil {
			return nil, err
		}

		// this part, doesn't allow for nested blocks
		if ast.Type != BlockStatement {
			body = append(body, *ast)
		} else {
			return nil, fmt.Errorf("ERROR: block isn't allowed inside of another block")
		}

	}

	tok = p.next()

	if tok.Kind != TokenCurlyBraceClose {
		return nil, fmt.Errorf("ERROR: expected a %v, got %v", TokenCurlyBraceClose, tok.Kind)
	}

	return &StatementNode{
		Type:     BlockStatement,
		Params:   args,
		Body:     body,
		Position: pos,
		Order:    p.Pos,
	}, nil
}

func (p *Parser) setHandler(pos Position) (*StatementNode, error) {
	args := make([]ExpressionNode, 0)
	tok := p.next()

	if tok.Kind != TokenIdentifier {
		return nil, fmt.Errorf("ERROR: expect a %v, got %v", TokenIdentifier, tok.Kind)
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
	// different types of token kind

	objPos := Position{}

	switch tok.Kind {
	case TokenString:
		path := tok.Text
		// check the param format
		// the param format needs to be a valid path
		if err := p.videoPathCheck(path); err != nil {
			return nil, err
		}
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
			Type:     IdentifierExpression,
			Value:    tok.Text,
			ExprType: IdentifierType,
			Position: Position{
				Row: tok.Row,
				Col: tok.Col,
			},
		})

	case TokenCurlyBraceOpen:

		objValue := make(ObjectLiteral)

		objPos.Col = tok.Col
		objPos.Row = tok.Row

		for p.peek().Kind != TokenCurlyBraceClose {
			key := p.next()

			if key.Kind != TokenIdentifier {
				return nil, fmt.Errorf("ERROR: expected a %v, got %v", TokenIdentifier, key.Kind)
			}

			objValue[key.Text] = ExpressionNode{}

			colon := p.next()

			if colon.Kind != TokenColon {
				return nil, fmt.Errorf("ERROR: expected a %v, got %v", TokenColon, colon.Kind)
			}

			value := p.next()

			var val ExpressionNode
			switch value.Kind {
			case TokenString:
				val = ExpressionNode{
					Type:     LiteralExpression,
					Value:    value.Text,
					ExprType: StringType,
					Position: Position{
						Row: key.Row,
						Col: key.Col,
					},
				}

			case TokenNumber:
				num, err := strconv.ParseFloat(value.Text, 64)
				if err != nil {
					return nil, fmt.Errorf("invalid number format, %v", err)
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
					Value:    value.Text == "true",
					ExprType: BooleanType,
					Position: Position{
						Row: key.Row,
						Col: key.Col,
					},
				}
			default:
				return nil, fmt.Errorf("ERROR: unsupported type %v", value.Kind)
			}

			objValue[key.Text] = val
		}

		args = append(args, ExpressionNode{
			Type:     ObjectExpression,
			Value:    objValue,
			ExprType: ObjectType,
			Position: objPos,
		})

		tok = p.next()

		if tok.Kind != TokenCurlyBraceClose {
			return nil, fmt.Errorf("ERROR: expected a %v, got %v", TokenCurlyBraceClose, tok.Kind)
		}

	default:
		return nil, fmt.Errorf("ERROR, %v isn't supportd, use (%v,%v,%v)", tok.Kind, TokenString, TokenIdentifier, TokenCurlyBraceOpen)
	}

	return &StatementNode{
		Type:     SetStatement,
		Params:   args,
		Position: pos,
		Order:    p.Pos,
	}, nil
}

func (p *Parser) useHandler(pos Position) (*StatementNode, error) {
	args := make([]ExpressionNode, 0)
	tok := p.next()

	if tok.Kind != TokenIdentifier {
		return nil, fmt.Errorf("ERROR: expect a %v, got %v", TokenIdentifier, tok.Kind)
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
		tok = p.next()

		if tok.Kind != TokenString && tok.Kind != TokenIdentifier {
			return nil, fmt.Errorf("ERROR: expect a (%v | %v), got %v", TokenString, TokenIdentifier, tok.Kind)
		}

		if tok.Kind == TokenString {
			if err := p.videoPathCheck(tok.Text); err != nil {
				return nil, err
			}

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
	args := make([]ExpressionNode, 0)

	tok := p.next()

	if tok.Kind != TokenString && tok.Kind != TokenIdentifier {
		return nil, fmt.Errorf("ERROR: expected %v, got %v", TokenString, tok.Kind)
	}
	// check the param format
	if tok.Kind == TokenString {
		path := tok.Text

		if err := p.videoPathCheck(path); err != nil {
			return nil, err
		}

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

func isValidPathFormat(path string) (bool, error) {
	if strings.ContainsAny(path, `<>:"|?*`) {
		return false, errors.New("ERROR: special characters like (<>:'|?*) are not allowed")
	}

	// Must be valid filepath format
	if !filepath.IsLocal(path) {
		return false, errors.New("ERROR: path is invalid")
	}

	// Must have extension
	ext := filepath.Ext(path)
	if ext == "" {
		return false, errors.New("ERROR: file at the end of path needs to have an extension")
	}
	return true, nil
}

type Mode = string

const (
	VIDEO Mode = "video"
	IMAGE Mode = "image"
)

func checkIfElementExist(slice []string, element string) bool {
	sort.Strings(slice)
	idx := sort.SearchStrings(slice, element)
	return idx < len(slice) && slice[idx] == element
}

func checkFileIsOfTypeMode(path string, mode Mode) bool {
	ext := filepath.Ext(path)

	modeOptions := make([]string, 0)
	switch mode {
	case VIDEO:
		modeOptions = videoExts
	case IMAGE:
		modeOptions = imageExts
	}

	return checkIfElementExist(modeOptions, ext)
}
