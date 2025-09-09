package parser_tests

import (
	"blk/lexer"
	"blk/parser"
	"fmt"
	"testing"
)

func TestAtomicVarDCL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			"let result: bool = true && false",
			"let result: bool = (true && false)",
		},
		{
			"let none_value: *i8 = nul",
			"let none_value: *i8 = nul",
		},
		{
			`let result: string = "Hello from " + "blk" `,
			`let result: string = ("Hello from " + "blk")`,
		},
		{
			"let result: f32 = 3.14 * 2.36 / 6.3",
			"let result: f32 = ((3.14 * 2.36) / 6.3)",
		},
		{
			"let result: i16 = 5 + 6 % 32",
			"let result: i16 = (5 + (6 % 32))",
		},
		// {
		// 	`let hash: map(string, struct{
		// 		username : string,
		// 		age : u8
		// 	}) = {}`,
		// 	// doesn't correctly output the expected format
		// 	`let hash: map(string, struct{ let username : string = , let age : u8 = }) = {}`,
		// },
		// {
		// 	`let hash = {
		// 		"hello" : [1 , 2],
		// 		"there" : [3 , 4]
		// 	}`,
		// 	`let hash = {"hello": [1,2], "there": [3,4]}`,
		// },
		{
			"result: bool = true && false",
			"let result: bool = (true && false)",
		},
		{
			"none_value :: nul",
			"const none_value = nul",
		},
		{
			`result: string = "Hello from" + "blk" `,
			`let result: string = ("Hello from" + "blk")`,
		},
		{
			"result :: 3.14 * 2.36 / 6.3",
			"const result = ((3.14 * 2.36) / 6.3)",
		},
		{
			"result :: 5 + 6 % 32",
			"const result = (5 + (6 % 32))",
		},
		{
			`hash := {}`,
			`let hash = {}`,
		},
	}

	for _, tt := range tests {
		l := lexer.NewLexer("", tt.input)
		p := parser.NewParser(l.Tokenize(), "")
		program := p.Parse()
		fmt.Println(program, tt.input)
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
			`Person :: struct {
				Name := "belkacem",
				Age := 22
			}`,
			`const Person = struct { let Name = "belkacem", let Age = 22,  }`,
		},
		{
			`User :: struct {
				Name := "lofi",
				getName :: fn(self) {
					return self.Name
				}
			}`,
			`const User = struct { let Name = "lofi", getName:fn(self){ return self.Name } }`,
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

func TestEnumDeclStatementDCL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			`Person :: enum {
				Child,
				Adult,
				Aged
			}`,
			`const Person = enum { Child, Adult, Aged }`,
		},
		{
			`Data :: enum {
   			Int,
    		Float,
    		String,
    		Bool
			}`,
			`const Data = enum { Int, Float, String, Bool }`,
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

func TestMemberShipAccessStatementDCL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			"result.Code = 200",
			"result.Code = 200",
		},
		{
			"file.meta.size = 2048",
			"file.meta.size = 2048",
		},
		{
			`response.body.userInfo.username = "John Doe"`,
			`response.body.userInfo.username = "John Doe"`,
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
