package tests

import (
	"blk/parser"
	"testing"
)

func TestFunctionParameterParsing(t *testing.T) {
	tests := []struct {
		input          string
		expectedParams []string
	}{
		{input: "fn void():void {}", expectedParams: []string{}},
		{input: "fn do_nothing(x:int):void {}", expectedParams: []string{"x"}},
		{input: "fn A3d(x : int, y : int, z:int): int {}", expectedParams: []string{"x", "y", "z"}},
	}
	for _, tt := range tests {
		l := parser.NewLexer("", tt.input)
		p := parser.NewParser(l.Tokenize(), "")
		program := p.Parse()
		functionStmt := program.Statements[0].(*parser.FunctionStatement)
		if len(functionStmt.Args) != len(tt.expectedParams) {
			t.Errorf("length parameters wrong. want %d, got=%d\n",
				len(tt.expectedParams), len(functionStmt.Args))
		}
	}
}
