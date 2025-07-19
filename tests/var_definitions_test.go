package tests

import (
	"blk/parser"
	"testing"
)

func TestAtomicLetStatementDCL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			"let result : bool = true && false",
			"let result = (true && false)",
		},
		{
			`let result: string = "Hello from " + "blk" `,
			`let result = ("Hello from" + "blk")`,
		},
		{
			"let result: float = 3.14 * 2.36 / 6.3",
			"let result = ((3.14 * 2.36) / 6.3)",
		},
		{
			"let result: int = 5 + 6 % 32",
			"let result = (5 + (6 % 32))",
		},
		{
			`var hash: map(string, array(int)) = {}`,
			`var hash = {}`,
		},
		{
			`var hash: map(string, array(int)) = {
				"hello" : [1 , 2],
				"there" : [3 , 4]
			}`,
			`var hash = {"hello": [1, 2], "there": [3, 4]}`,
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

func TestStructLetStatementDCL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			`struct Person {
				Name : string
				Age : int
			}`,
			"struct Person { Name:string, Age:int }",
		},
		{
			`struct Person {
				Name : string
				Age : int
				Child: Person
			}`,
			"struct Person { Name:string, Age:int, Child:Person }",
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

func TestTypeStatementDCL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			"type Martix = array(array(int))",
			"type Martix = array(array(int))",
		},
		{
			"type Session = map(string, User)",
			"type Session = map(string, User)",
		},
		{
			"type FullName = [2]string",
			"type FullName = [2]string",
		},
		{
			"type WhoCreatesTypesLikeThis = [2]array([1]int)",
			"type WhoCreatesTypesLikeThis = [2]array([1]int)",
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

func TestMemberShipAccessStatementDCL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			"result.Code = 200",
			"result.(Code = 200)",
		},
		{
			"file.meta.size = 2048",
			"file.meta.(size = 2048)",
		},
		{
			`response.body.userInfo.username = "John Doe"`,
			`response.body.userInfo.(username = "John Doe")`,
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
