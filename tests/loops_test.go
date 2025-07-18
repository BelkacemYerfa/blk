package tests

import (
	"blk/parser"
	"testing"
)

func TestWhileLoopsStatments(t *testing.T) {
	tests := []struct {
		input          string
		expected       string
	}{
		{
			input: `while i <= n {
				sum = sum + i
				i = i + 1
			}`,
			expected: "while (i <= n) { (sum = (sum + i))(i = (i + 1)) }",
		},
		{
			input: `while i <= n {
				if i > 10 {
					print(i)
				}
				i = i + 1
			}`,
			expected: "while (i <= n) { if (i > 10) { print(i) }(i = (i + 1)) }",
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
