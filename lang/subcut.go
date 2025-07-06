package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
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

func checkIfElementExist(slice []string, element string) bool {
	sort.Strings(slice)
	idx := sort.SearchStrings(slice, element)
	return idx < len(slice) && slice[idx] == element
}

func checkTimeTrimFormatValid(tm string) bool {
	_, err := time.Parse("15:04:05", tm)
	if err != nil {
		return false
	}
	return true
}

func main() {
	osPath, _ := os.Getwd()

	path := filepath.Join(osPath, "lang/examples/main.subcut")

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

	lexer := NewLexer(path, content)
	tokens := lexer.Tokenize()

	// Write tokens to file
	// Write tokens to JSON file
	tokensFile := filepath.Join(osPath, "lang/examples/main_tokens.json")
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

	parser := NewParser(tokens)
	ast := parser.Parse(lexer)

	// Write AST to JSON file
	astFile := filepath.Join(osPath, "lang/examples/main_ast.json")
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
