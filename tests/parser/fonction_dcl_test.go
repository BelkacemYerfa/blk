package parser_tests

import (
	"blk/ast"
	"blk/lexer"
	"blk/parser"
	"testing"
)

func TestFunctionParameterParsing(t *testing.T) {
	tests := []struct {
		input          string
		expectedParams []string
	}{
		{input: "void :: fn () {}", expectedParams: []string{}},
		{input: "do_nothing :: fn(x) {}", expectedParams: []string{"x"}},
		{input: "A3d :: fn(x, y, z) {}", expectedParams: []string{"x", "y", "z"}},
	}
	for _, tt := range tests {
		l := lexer.NewLexer("", tt.input)
		p := parser.NewParser(l, "")
		program := p.Parse()
		functionStmt := program.Statements[0].(*ast.VarDeclaration).Value.(*ast.FunctionExpression)

		if len(functionStmt.Args) != len(tt.expectedParams) {
			t.Errorf("length parameters wrong. want %d, got=%d\n",
				len(tt.expectedParams), len(functionStmt.Args))
		}
	}
}
