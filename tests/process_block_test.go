package tests

import (
	"subcut/parser"
	"testing"

	"github.com/go-test/deep"
)

func TestSimpleProcess(t *testing.T) {
	code := `
process first_pipe {
	# empty
}
		`

	output := &[]parser.StatementNode{
		{
			Type: parser.ProcessStatement,
			Params: []any{
				parser.ExpressionNode{
					Type:     parser.IdentifierExpression,
					Value:    "first_pipe",
					ExprType: parser.IdentifierType,
					Position: parser.Position{
						Col: 9,
						Row: 2,
					},
				},
			},
			Body: []parser.StatementNode{},
			Position: parser.Position{
				Col: 1,
				Row: 2,
			},
			// order takes the cursor position
			Order: 4,
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

func TestNestedProcess(t *testing.T) {
	code := `
process first_pipe {
	process nested_pipe {
		# empty
	}
}
		`

	output := (*[]parser.StatementNode)(nil)

	lexer := parser.NewLexer("", code)
	tokens := lexer.Tokenize()

	p := parser.NewParser(tokens, "")
	ast := p.Parse()

	if ast != nil {
		t.Errorf("ERROR: ast is not nil")
	}

	if diff := deep.Equal(ast, output); diff != nil {
		t.Error(diff)
	}
}
