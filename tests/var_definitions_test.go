package tests

import (
	"subcut/parser"
	"testing"

	"github.com/go-test/deep"
)

func TestSimpleSetVarDeclaration(t *testing.T) {
	code := `
set input "/assets/smt.mp4"
		`

	output := &[]parser.StatementNode{
		{
			Type: parser.SetStatement,
			Params: []any{
				parser.ExpressionNode{
					Type:       parser.IdentifierExpression,
					Identifier: "input",
					Value:      "/assets/smt.mp4",
					ExprType:   parser.IdentifierType,
					Position: parser.Position{
						Col: 5,
						Row: 2,
					},
				},
			},
			Position: parser.Position{
				Col: 1,
				Row: 2,
			},
			// order takes the cursor position
			Order: 3,
		},
	}

	lexer := parser.NewLexer("", code)
	tokens := lexer.Tokenize()

	p := parser.NewParser(tokens, "")
	ast := p.Parse()

	if ast == nil {
		t.Errorf("ERROR: ast is nil")
		return
	}

	if diff := deep.Equal(ast, output); diff != nil {
		t.Error(diff)
	}
}

func TestSimpleNumberSetVarDeclaration(t *testing.T) {
	code := `
set input 3
set record -0.3
		`

	output := &[]parser.StatementNode{
		{
			Type: parser.SetStatement,
			Params: []any{
				parser.ExpressionNode{
					Type:       parser.IdentifierExpression,
					Identifier: "input",
					Value:      float64(3),
					ExprType:   parser.IdentifierType,
					Position: parser.Position{
						Col: 5,
						Row: 2,
					},
				},
			},
			Position: parser.Position{
				Col: 1,
				Row: 2,
			},
			// order takes the cursor position
			Order: 3,
		},
		{
			Type: parser.SetStatement,
			Params: []any{
				parser.ExpressionNode{
					Type:       parser.IdentifierExpression,
					Identifier: "record",
					Value:      float64(-0.3),
					ExprType:   parser.IdentifierType,
					Position: parser.Position{
						Col: 5,
						Row: 3,
					},
				},
			},
			Position: parser.Position{
				Col: 1,
				Row: 3,
			},
			// order takes the cursor position
			Order: 7,
		},
	}

	lexer := parser.NewLexer("", code)
	tokens := lexer.Tokenize()

	p := parser.NewParser(tokens, "")
	ast := p.Parse()

	if ast == nil {
		t.Errorf("ERROR: ast is nil")
		return
	}

	if diff := deep.Equal(ast, output); diff != nil {
		t.Error(diff)
	}
}

func TestSimpleObjectSetVarDeclaration(t *testing.T) {
	code := `
set bg_track {
	path : "/assets/smt.mp4"
	duck : true
}
`

	output := &[]parser.StatementNode{
		{
			Type: parser.SetStatement,
			Params: []any{
				parser.ExpressionNode{
					Type:       parser.IdentifierExpression,
					Identifier: "bg_track",
					Value: parser.ObjectLiteral{
						"path": parser.ExpressionNode{
							Type:     parser.LiteralExpression,
							Value:    "/assets/smt.mp4",
							ExprType: parser.StringType,
							Position: parser.Position{
								Col: 2,
								Row: 3,
							},
						},
						"duck": parser.ExpressionNode{
							Type:     parser.LiteralExpression,
							Value:    true,
							ExprType: parser.BooleanType,
							Position: parser.Position{
								Col: 2,
								Row: 4,
							},
						},
					},
					ExprType: parser.IdentifierType,
					Position: parser.Position{
						Col: 5,
						Row: 2,
					},
				},
			},
			Position: parser.Position{
				Col: 1,
				Row: 2,
			},
			// order takes the cursor position
			Order: 10,
		},
	}

	lexer := parser.NewLexer("", code)
	tokens := lexer.Tokenize()

	p := parser.NewParser(tokens, "")
	ast := p.Parse()

	if ast == nil {
		t.Errorf("ERROR: ast is nil")
		return
	}

	if diff := deep.Equal(ast, output); diff != nil {
		t.Error(diff)
	}
}
