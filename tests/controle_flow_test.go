package tests

import (
	"blk/parser"
	"testing"
)

func TestIfExpressionsParsing(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "if a > b {}",
			expected: "if (a > b) {  }",
		},
		{
			input:    "if a + b < 0 {} else { return a }",
			expected: "if ((a + b) < 0) {  } else { return a }",
		},
		{
			input:    "if a < 0 {} else if a > 0 && true { return a } else { return a - 1 }",
			expected: "if (a < 0) {  } else if ((a > 0) && true) { return a } else { return (a - 1) }",
		},
	}
	for _, tt := range tests {
		l := parser.NewLexer("", tt.input)
		p := parser.NewParser(l.Tokenize(), "")
		program := p.Parse()
		actual := program.String()
		if actual != tt.expected {
			t.Errorf("expected=%q, got=%q", tt.expected, actual)
		}
	}
}
