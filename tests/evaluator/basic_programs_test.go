package evaluator_tests

import (
	"blk/interpreter"
	"blk/lexer"
	"blk/object"
	"blk/parser"
	"testing"
)

func TestFuncEvaluation(t *testing.T) {
	tests := []struct {
		input    string
		expected object.Object
	}{
		{
			input: `
fact :: fn(n) {
  if n <= 1 { 1 } else { n * fact(n - 1) }
}

res :: fact(5)
res
`,
			expected: &object.Integer{Value: 120},
		},
		{
			input: `
prod :: fn(n) {
  return n*n
}

res :: prod(5)
res
`,
			expected: &object.Integer{Value: 25},
		},
	}
	for _, tt := range tests {
		l := lexer.NewLexer("", tt.input)
		p := parser.NewParser(l.Tokenize(), "")
		program := p.Parse()
		evaluator := interpreter.NewInterpreter(nil, "")
		eval := evaluator.Eval(program)
		if eval == nil {
			t.Errorf("evaluation is null")
		}
		if eval.Inspect() != tt.expected.Inspect() {
			t.Errorf("expected=%q, got=%q", tt.expected, eval.Inspect())
		}
	}
}

func TestIfEvaluation(t *testing.T) {
	tests := []struct {
		input    string
		expected object.Object
	}{
		{
			input: `
res :: if true {
			"Hello"
} else {
			"See ya"
}
res
`,
			expected: &object.String{Value: "Hello"},
		},
		{
			input: `
res :: if false ? "Hello" : "See ya"
res
`,
			expected: &object.String{Value: "See ya"},
		},
		{
			input: `
res :: if false use "Hello" else "See ya"
res
`,
			expected: &object.String{Value: "See ya"},
		},
	}
	for _, tt := range tests {
		l := lexer.NewLexer("", tt.input)
		p := parser.NewParser(l.Tokenize(), "")
		program := p.Parse()
		evaluator := interpreter.NewInterpreter(nil, "")
		eval := evaluator.Eval(program)
		if eval == nil {
			t.Errorf("evaluation is null")
		}
		if eval.Inspect() != tt.expected.Inspect() {
			t.Errorf("expected=%q, got=%q", tt.expected, eval.Inspect())
		}
	}
}

func TestArrayOp(t *testing.T) {
	tests := []struct {
		input    string
		expected object.Object
	}{
		{
			input: `
res :: [1,2,3]
res[0]+3
`,
			expected: &object.Integer{Value: 4},
		},
		{
			input: `
res :: [1,2,3]
res[0]*3-9+6
`,
			expected: &object.Integer{Value: 0},
		},
		{
			input: `
res :: [1,2,3]
r :: res[0]*res[1]*res[2]
string(r)
`,
			expected: &object.String{Value: "6"},
		},
	}
	for _, tt := range tests {
		l := lexer.NewLexer("", tt.input)
		p := parser.NewParser(l.Tokenize(), "")
		program := p.Parse()
		evaluator := interpreter.NewInterpreter(nil, "")
		eval := evaluator.Eval(program)
		if eval == nil {
			t.Errorf("evaluation is null")
		}
		if eval.Inspect() != tt.expected.Inspect() {
			t.Errorf("expected=%q, got=%q", tt.expected, eval.Inspect())
		}
	}
}
