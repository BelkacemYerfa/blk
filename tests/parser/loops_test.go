package parser_tests

import (
	"blk/lexer"
	"blk/parser"
	"testing"
)

func TestLoopsStatments(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input: `while i <= n {
				sum = sum + i
				i = i + 1
			}`,
			expected: "while (i <= n) { sum = (sum + i)i = (i + 1) }",
		},
		{
			input: `while i <= n {
				if i > 10 {
					print(i)
				}
				i = i + 1
			}`,
			expected: "while (i <= n) { if (i > 10) { print(i) }i = (i + 1) }",
		},
		{
			input: `for i, val in ["hi", "hello"] {
				print(i)
			}`,
			expected: `for i, val in ["hi","hello"] { print(i) }`,
		},
		{
			input: `for key, value in Users {
				print(key, value)
			}`,
			expected: `for key, value in Users { print(key, value) }`,
		},
		{
			input: `for key, value in 0..=10 {
				print(key, value)
			}`,
			expected: `for key, value in 0..=10 { print(key, value) }`,
		},
		{
			input: `for key, value in 0..=len(input) {
				print(key, value)
			}`,
			expected: `for key, value in 0..=len(input) { print(key, value) }`,
		},
	}
	for _, tt := range tests {
		l := lexer.NewLexer("", tt.input)
		p := parser.NewParser(l.Tokenize(), "")
		program := p.Parse()
		actual := program.String()
		if actual != tt.expected {
			t.Errorf("expected=%q, got=%q", tt.expected, actual)
		}
	}
}
