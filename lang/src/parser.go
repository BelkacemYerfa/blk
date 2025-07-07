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
	SET_VIDEO      Command = "set_video"
	SET_TRACK      Command = "set_track"
	USE_TRACK      Command = "use_track"
	BLOCK          Command = "block"
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

type Param struct {
	Kind  TokenKind
	Value any
	Row   int
	Col   int
}

type ASTNode struct {
	Command Command
	Params  []Param
	Body    AST
	Order   int
}

type AST = []ASTNode

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
		fmt.Println(tok)
		switch tok.Kind {
		case TokenPush, TokenConcat, TokenTrim, TokenExport, TokenSetTrack, TokenUseTrack, TokenThumbnailFrom, TokenSetVideo, TokenBlock:
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

func (p *Parser) parseCommand() (*ASTNode, error) {
	cmdToken := p.next() // Consume command

	// Validation step
	switch cmdToken.Kind {
	case TokenPush:
		return p.pushHandler()
	case TokenTrim:
		return p.trimHandler()
	case TokenConcat:
		return p.concatHandler()
	case TokenThumbnailFrom:
		return p.thumbnailHandler()
	case TokenExport:
		return p.exportHandler()
	case TokenSetTrack:
		return p.setTrackHandler()
	case TokenUseTrack:
		return p.useTrackHandler()
	case TokenSetVideo:
		return p.setVideoHandler()
	case TokenBlock:
		return p.blockHandler()
	}

	// All good, create AST node
	return &ASTNode{}, fmt.Errorf("ERROR: unexpected token appeared, line %v row%v", cmdToken.Row, cmdToken.Col)
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

func (p *Parser) pushHandler() (*ASTNode, error) {
	args := make([]Param, 0)

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
	}

	args = append(args, Param{
		Value: tok.Text,
		Kind:  tok.Kind,
		Row:   tok.Row,
		Col:   tok.Col,
	})

	return &ASTNode{
		Command: PUSH,
		Params:  args,
		Order:   p.Pos,
	}, nil
}

func (p *Parser) trimHandler() (*ASTNode, error) {
	args := make([]Param, 0)

	for range 2 {
		tok := p.next()
		if tok.Kind != TokenTime {
			return nil, fmt.Errorf("ERROR: expected %v, got %v", TokenTime, tok.Kind)
		}

		args = append(args, Param{
			Value: tok.Text,
			Kind:  TokenTime,
			Row:   tok.Row,
			Col:   tok.Col,
		})
	}

	// check the format of the path if it exists
	tok := p.next()

	videoTarget := "all"

	switch tok.Kind {
	case TokenString:
		if err := p.videoPathCheck(tok.Text); err != nil {
			return nil, err
		}

		videoTarget = tok.Text
	case TokenIdentifier:
		videoTarget = tok.Text
	default:
		p.Pos--
	}

	args = append(args, Param{
		Value: videoTarget,
		Kind:  TokenString,
		Row:   tok.Row,
		Col:   tok.Col,
	})

	return &ASTNode{
		Command: TRIM,
		Params:  args,
		Order:   p.Pos,
	}, nil
}

func (p *Parser) concatHandler() (*ASTNode, error) {
	tok := p.peek()

	if tokenKey, isMatched := keywords[tok.Text]; isMatched {
		if tokenKey == TokenBool {
			return nil, fmt.Errorf("ERROR: expected nothing, got %v", TokenBool)
		}
	}
	return &ASTNode{
		Command: CONCAT,
		Params:  []Param{},
		Order:   p.Pos,
	}, nil
}

func (p *Parser) thumbnailHandler() (*ASTNode, error) {
	args := make([]Param, 0)

	tok := p.next()

	format := tok.Text

	switch tok.Kind {
	case TokenTime:
		timeFormat := `^\d{2}:\d{2}:\d{2}$`
		format := tok.Text
		if matched, _ := regexp.MatchString(timeFormat, format); matched {

			args = append(args, Param{
				Value: tok.Text,
				Kind:  TokenTime,
				Row:   tok.Row,
				Col:   tok.Col,
			})
		}
	case TokenNumber:
		num, err := strconv.Atoi(format)
		if err != nil {
			return nil, fmt.Errorf("invalid number format, %v", err)
		}
		args = append(args, Param{
			Value: num,
			Kind:  TokenNumber,
			Row:   tok.Row,
			Col:   tok.Col,
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
	}

	args = append(args, Param{
		Value: tok.Text,
		Kind:  tok.Kind,
		Row:   tok.Row,
		Col:   tok.Col,
	})

	return &ASTNode{
		Command: THUMBNAIL_FROM,
		Params:  args,
		Order:   p.Pos,
	}, nil
}

func (p *Parser) blockHandler() (*ASTNode, error) {
	args := make([]Param, 0)
	tok := p.next()

	if tok.Kind != TokenIdentifier {
		return nil, fmt.Errorf("ERROR: expect a %v, got %v", TokenIdentifier, tok.Kind)
	}

	args = append(args, Param{
		Value: tok.Text,
		Kind:  TokenIdentifier,
		Row:   tok.Row,
		Col:   tok.Col,
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
		if ast.Command != BLOCK {
			body = append(body, *ast)
		} else {
			return nil, fmt.Errorf("ERROR: block isn't allowed inside of another block")
		}

	}

	tok = p.next()

	if tok.Kind != TokenCurlyBraceClose {
		return nil, fmt.Errorf("ERROR: expected a %v, got %v", TokenCurlyBraceClose, tok.Kind)
	}

	return &ASTNode{
		Command: BLOCK,
		Params:  args,
		Body:    body,
		Order:   p.Pos,
	}, nil
}

func (p *Parser) setVideoHandler() (*ASTNode, error) {
	args := make([]Param, 0)
	tok := p.next()

	if tok.Kind != TokenIdentifier {
		return nil, fmt.Errorf("ERROR: expect a %v, got %v", TokenIdentifier, tok.Kind)
	}

	args = append(args, Param{
		Value: tok.Text,
		Kind:  TokenIdentifier,
		Row:   tok.Row,
		Col:   tok.Col,
	})

	tok = p.next()

	if tok.Kind != TokenString {
		return nil, fmt.Errorf("ERROR: unexpected value at line %v, row %v\nvalue needs to be string", tok.Row, tok.Col)
	}

	path := tok.Text

	// check the param format
	// the param format needs to be a valid path
	if err := p.videoPathCheck(path); err != nil {
		return nil, err
	}

	args = append(args, Param{
		Value: tok.Text,
		Kind:  TokenString,
		Row:   tok.Row,
		Col:   tok.Col,
	})

	return &ASTNode{
		Command: SET_VIDEO,
		Params:  args,
		Order:   p.Pos,
	}, nil
}

func (p *Parser) setTrackHandler() (*ASTNode, error) {
	args := make([]Param, 0)
	tok := p.next()

	if tok.Kind != TokenIdentifier {
		return nil, fmt.Errorf("ERROR: expect a %v, got %v", TokenIdentifier, tok.Kind)
	}

	args = append(args, Param{
		Value: tok.Text,
		Kind:  TokenIdentifier,
		Row:   tok.Row,
		Col:   tok.Col,
	})

	tok = p.next()

	if tok.Kind != TokenCurlyBraceOpen {
		return nil, fmt.Errorf("ERROR: expected a %v, got %v", TokenCurlyBraceOpen, tok.Kind)
	}

	for p.peek().Kind != TokenCurlyBraceClose {
		key := p.next()

		if key.Kind != TokenIdentifier {
			return nil, fmt.Errorf("ERROR: expected a %v, got %v", TokenIdentifier, key.Kind)
		}

		args = append(args, Param{
			Value: key.Text,
			Kind:  TokenIdentifier,
			Row:   key.Row,
			Col:   key.Col,
		})

		colon := p.next()

		if colon.Kind != TokenColon {
			return nil, fmt.Errorf("ERROR: expected a %v, got %v", TokenColon, colon.Kind)
		}

		value := p.next()

		var val Param
		switch value.Kind {
		case TokenString:
			val = Param{
				Value: value.Text,
				Kind:  TokenString,
				Row:   value.Row,
				Col:   value.Col,
			}

		case TokenNumber:
			num, err := strconv.ParseFloat(value.Text, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid number format, %v", err)
			}
			val = Param{
				Value: num,
				Kind:  TokenNumber,
				Row:   value.Row,
				Col:   value.Col,
			}
		case TokenBool:
			val = Param{
				Value: value.Text == "true",
				Kind:  TokenBool,
				Row:   value.Row,
				Col:   value.Col,
			}
		default:
			return nil, fmt.Errorf("ERROR: unsupported type %v", value.Kind)
		}

		args = append(args, val)
	}

	tok = p.next()

	if tok.Kind != TokenCurlyBraceClose {
		return nil, fmt.Errorf("ERROR: expected a %v, got %v", TokenCurlyBraceClose, tok.Kind)
	}

	return &ASTNode{
		Command: SET_TRACK,
		Params:  args,
		Order:   p.Pos,
	}, nil
}

func (p *Parser) useTrackHandler() (*ASTNode, error) {
	args := make([]Param, 0)
	tok := p.next()

	if tok.Kind != TokenIdentifier {
		return nil, fmt.Errorf("ERROR: expect a %v, got %v", TokenIdentifier, tok.Kind)
	}

	args = append(args, Param{
		Value: tok.Text,
		Kind:  TokenIdentifier,
		Row:   tok.Row,
		Col:   tok.Col,
	})

	tok = p.next()

	if tok.Kind != TokenString && tok.Kind != TokenIdentifier {
		return nil, fmt.Errorf("ERROR: expected %v, got %v", TokenString, tok.Kind)
	}
	// check the param format

	if tok.Kind == TokenString {
		path := tok.Text

		if err := p.videoPathCheck(path); err != nil {
			return nil, err
		}
	}

	args = append(args, Param{
		Value: tok.Text,
		Kind:  tok.Kind,
		Row:   tok.Row,
		Col:   tok.Col,
	})

	return &ASTNode{
		Command: USE_TRACK,
		Params:  args,
		Order:   p.Pos,
	}, nil
}

func (p *Parser) exportHandler() (*ASTNode, error) {
	args := make([]Param, 0)

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
	}

	args = append(args, Param{
		Value: tok.Text,
		Kind:  tok.Kind,
		Row:   tok.Row,
		Col:   tok.Col,
	})

	return &ASTNode{
		Command: EXPORT,
		Params:  args,
		Order:   p.Pos,
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
