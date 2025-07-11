package tests

import (
	"subcut/parser"
	"testing"

	"github.com/go-test/deep"
)

func TestSimpleIfDeclaration(t *testing.T) {
	code := `
if luck == 3 {
	# empty
}
	`

	output := &[]parser.StatementNode{
		{
			Type: parser.IfStatement,
			Params: []any{
				parser.BinaryExpressionNode{
					Type:     parser.BinaryExpression,
					Operator: "==",
					Left: parser.ExpressionNode{
						Type: parser.IdentifierExpression,
						Value: &parser.MemberAccessExpression{
							Name:     "luck",
							Property: nil,
						},
						ExprType: parser.IdentifierType,
						Position: parser.Position{
							Col: 4,
							Row: 2,
						},
					},
					Right: parser.ExpressionNode{
						Type:     parser.LiteralExpression,
						Value:    float64(3),
						ExprType: parser.NumberType,
						Position: parser.Position{
							Col: 12,
							Row: 2,
						},
					},
				},
			},
			Body: []parser.StatementNode{},
			Position: parser.Position{
				Col: 1,
				Row: 2,
			},
			// order takes the cursor position
			Order: 6,
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

func TestSimpleIfElseDeclaration(t *testing.T) {
	code := `
if luck == 3 {
	# empty
} else {
	# empty
}
	`

	output := &[]parser.StatementNode{
		{
			Type: parser.IfStatement,
			Params: []any{
				parser.BinaryExpressionNode{
					Type:     parser.BinaryExpression,
					Operator: "==",
					Left: parser.ExpressionNode{
						Type: parser.IdentifierExpression,
						Value: &parser.MemberAccessExpression{
							Name:     "luck",
							Property: nil,
						},
						ExprType: parser.IdentifierType,
						Position: parser.Position{
							Col: 4,
							Row: 2,
						},
					},
					Right: parser.ExpressionNode{
						Type:     parser.LiteralExpression,
						Value:    float64(3),
						ExprType: parser.NumberType,
						Position: parser.Position{
							Col: 12,
							Row: 2,
						},
					},
				},
			},
			Body: []parser.StatementNode{},
			Position: parser.Position{
				Col: 1,
				Row: 2,
			},
			// order takes the cursor position
			Order: 6,
		},
		{
			Type:   parser.ElseStatement,
			Params: []any{},
			Body:   []parser.StatementNode{},
			Position: parser.Position{
				Col: 3,
				Row: 4,
			},
			Order: 9,
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

func TestMultiIfElseDeclaration(t *testing.T) {
	code := `
if luck == 3 {
	# empty
} else if luck < 2 {
	# empty
} else {
	# empty
}
	`

	output := &[]parser.StatementNode{
		{
			Type: parser.IfStatement,
			Params: []any{
				parser.BinaryExpressionNode{
					Type:     parser.BinaryExpression,
					Operator: "==",
					Left: parser.ExpressionNode{
						Type: parser.IdentifierExpression,
						Value: &parser.MemberAccessExpression{
							Name:     "luck",
							Property: nil,
						},
						ExprType: parser.IdentifierType,
						Position: parser.Position{
							Col: 4,
							Row: 2,
						},
					},
					Right: parser.ExpressionNode{
						Type:     parser.LiteralExpression,
						Value:    float64(3),
						ExprType: parser.NumberType,
						Position: parser.Position{
							Col: 12,
							Row: 2,
						},
					},
				},
			},
			Body: []parser.StatementNode{},
			Position: parser.Position{
				Col: 1,
				Row: 2,
			},
			// order takes the cursor position
			Order: 6,
		},
		{
			Type: parser.IfStatement,
			Params: []any{
				parser.BinaryExpressionNode{
					Type:     parser.BinaryExpression,
					Operator: "<",
					Left: parser.ExpressionNode{
						Type: parser.IdentifierExpression,
						Value: &parser.MemberAccessExpression{
							Name:     "luck",
							Property: nil,
						},
						ExprType: parser.IdentifierType,
						Position: parser.Position{
							Col: 11,
							Row: 4,
						},
					},
					Right: parser.ExpressionNode{
						Type:     parser.LiteralExpression,
						Value:    float64(2),
						ExprType: parser.NumberType,
						Position: parser.Position{
							Col: 18,
							Row: 4,
						},
					},
				},
			},
			Body: []parser.StatementNode{},
			Position: parser.Position{
				Col: 3,
				Row: 4,
			},
			// order takes the cursor position
			Order: 13,
		},
		{
			Type:   parser.ElseStatement,
			Params: []any{},
			Body:   []parser.StatementNode{},
			Position: parser.Position{
				Col: 3,
				Row: 6,
			},
			Order: 16,
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

func TestSimpleIfDeclarationWithNegativeNumbers(t *testing.T) {
	code := `
if luck == -3 {
	# empty
}
	`

	output := &[]parser.StatementNode{
		{
			Type: parser.IfStatement,
			Params: []any{
				parser.BinaryExpressionNode{
					Type:     parser.BinaryExpression,
					Operator: "==",
					Left: parser.ExpressionNode{
						Type: parser.IdentifierExpression,
						Value: &parser.MemberAccessExpression{
							Name:     "luck",
							Property: nil,
						},
						ExprType: parser.IdentifierType,
						Position: parser.Position{
							Col: 4,
							Row: 2,
						},
					},
					Right: parser.ExpressionNode{
						Type:     parser.LiteralExpression,
						Value:    -float64(3),
						ExprType: parser.NumberType,
						Position: parser.Position{
							Col: 13,
							Row: 2,
						},
					},
				},
			},
			Body: []parser.StatementNode{},
			Position: parser.Position{
				Col: 1,
				Row: 2,
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
