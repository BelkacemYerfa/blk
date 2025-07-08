package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"subcut/src"
	"subcut/tests"
)

func checkPathExistence(path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, errors.New("ERROR: provided file doesn't exist, please check your path")
		} else {
			fmt.Println(err)
			return false, errors.New("ERROR: accessing the file")
		}
	}

	return true, nil
}

func dev() {
	osPath, _ := os.Getwd()

	path := filepath.Join(osPath, "./internal_examples/main.subcut")

	valid, err := checkPathExistence(path)

	if err != nil && !valid {
		fmt.Println(err)
		return
	}

	ext := filepath.Ext(path)

	if ext != ".subcut" {
		fmt.Println("ERROR: please provide a file with subcut extension")
		return
	}

	byteCtn, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}

	content := string(byteCtn)

	lexer := src.NewLexer(path, content)
	tokens := lexer.Tokenize()

	// Write tokens to file
	// Write tokens to JSON file
	tokensFile := filepath.Join(osPath, "./internal_examples/main_tokens.json")
	tokensJSON, err := json.Marshal(tokens)
	if err != nil {
		fmt.Printf("ERROR marshaling tokens to JSON: %v\n", err)
		return
	}
	err = os.WriteFile(tokensFile, tokensJSON, 0644)
	if err != nil {
		fmt.Printf("ERROR writing tokens file: %v\n", err)
		return
	}

	parser := src.NewParser(tokens)
	ast := parser.Parse()

	// Write AST to JSON file
	astFile := filepath.Join(osPath, "./internal_examples/main_ast.json")
	astJSON, err := json.Marshal(ast)
	if err != nil {
		fmt.Printf("ERROR marshaling AST to JSON: %v\n", err)
		return
	}
	err = os.WriteFile(astFile, astJSON, 0644)
	if err != nil {
		fmt.Printf("ERROR writing AST file: %v\n", err)
		return
	}
}

func main() {

	arg := ""

	if len(os.Args) < 1 {
		arg = "dev"
	} else {
		arg = os.Args[1]
	}

	if arg == "dev" {
		dev()
	} else {
		tests.TestRunner()
	}
}
