package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"
)

type TOKEN = string

const (
	PUSH   TOKEN = "push"
	TRIM   TOKEN = "trim"
	EXPORT TOKEN = "export"
	CONCAT TOKEN = "concat"
)

var (
	videoExts = []string{
		".mp4", ".mov", ".avi", ".mkv",
		".webm", ".flv", ".wmv",
	}
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

func createKey(command string, line uint) string {
	return fmt.Sprintf("%s@%d", command, line)
}

func checkIfElementExist(slice []string, element string) bool {
	sort.Strings(slice)
	idx := sort.SearchStrings(slice, element)
	return idx < len(videoExts) && videoExts[idx] == element
}

func fileIsVideo(path string) bool {
	ext := filepath.Ext(path)

	return checkIfElementExist(videoExts, ext)
}

func checkTimeTrimFormatValid(tm string) bool {
	_, err := time.Parse("hh:mm:ss", tm)
	if err != nil {
		return false
	}
	return true
}

type Instruction struct {
	Params []string
	Order  uint
}

type Lexer struct {
	Tokens map[string]Instruction
	Line   uint
}

func NewLexer() *Lexer {
	return &Lexer{
		Tokens: make(map[string]Instruction),
		Line:   0,
	}
}

func (l *Lexer) Tokenize(content string) {
	chunks := strings.Split(content, "\n")

	for idx, chunk := range chunks {
		list := strings.Split(chunk, " ")

		operation := list[0]
		params := list[1:]

		params = slices.DeleteFunc(params, func(param string) bool {
			return len(strings.TrimSpace(param)) == 0
		})

		l.Line = uint(idx) + 1

		switch strings.ToLower(operation) {
		case PUSH:
			if err := l.pushHandler(idx, params); err != nil {
				fmt.Printf(
					"%v\nLine: %v\nInstruction: %s", err, idx+1, chunk,
				)
				return
			}
		case TRIM:
			if err := l.trimHandler(idx, params); err != nil {
				fmt.Printf(
					"%v\nLine: %v\nInstruction: %s", err, idx+1, chunk,
				)
				return
			}
		case CONCAT:
			if err := l.concatHandler(idx, params); err != nil {
				fmt.Printf(
					"%v\nLine: %v\nInstruction: %s", err, idx+1, chunk,
				)
				return
			}
		case EXPORT:
			if err := l.exportHandler(idx, params); err != nil {
				fmt.Printf(
					"%v\nLine: %v\nInstruction: %s", err, idx+1, chunk,
				)
				return
			}

		}
	}
}

func (l *Lexer) addToken(key string, value Instruction) {
	if existing, ok := l.Tokens[key]; ok {
		existing.Params = append(existing.Params, value.Params...)
		l.Tokens[key] = existing
	} else {
		l.Tokens[key] = value
	}
}

func (l *Lexer) pushHandler(idx int, params []string) error {
	if len(params) > 1 {
		return errors.New("ERROR: push can't have more than one param")
	}

	if len(params) < 1 {
		return errors.New("ERROR: push can't have less than one param")
	}

	// check the param format
	// the param format needs to be a valid path
	path := params[0]

	path = strings.TrimSpace(path)

	osPath, _ := os.Getwd()
	path = filepath.Join(osPath, path)

	_, err := checkPathExistence(path)

	if err != nil {
		return err
	}

	key := createKey(PUSH, l.Line)
	value := Instruction{
		Params: []string{path},
		Order:  uint(idx),
	}
	l.addToken(key, value)

	return nil
}

func (l *Lexer) trimHandler(idx int, params []string) error {
	if len(params) < 2 {
		return errors.New("ERROR: trim can't have less than two params <start-time> <end-time>")
	}

	if len(params) > 3 {
		return errors.New("ERROR: trim can't have more than three params <start-time> <end-time> <video-target>?")
	}

	// check the format of the path if it exists
	videoTarget := "all"
	if len(params) == 3 {
		videoTarget = params[2]
		params = params[:2]
	}

	if videoTarget != "all" {
		videoTarget = strings.TrimSpace(videoTarget)

		osPath, _ := os.Getwd()
		videoTarget = filepath.Join(osPath, videoTarget)
	}

	// check the time format of both start and end

	for _, param := range params {
		if checkTimeTrimFormatValid(param) {
			return fmt.Errorf("ERROR: %v isn't in valid format to be used as time", param)
		}
	}

	key := createKey(TRIM, l.Line)
	trimParams := append([]string{videoTarget}, params[:2]...)
	value := Instruction{
		Params: trimParams,
		Order:  uint(idx),
	}
	l.addToken(key, value)
	return nil
}

func (l *Lexer) concatHandler(idx int, params []string) error {
	if len(params) > 0 {
		return errors.New("ERROR: concat doesn't have params")
	}

	if len(l.Tokens) < 2 {
		return errors.New("ERROR: can't use concat without having 2 videos in the tokens")
	}

	key := createKey(CONCAT, l.Line)
	value := Instruction{
		Params: []string{},
		Order:  uint(idx),
	}
	l.addToken(key, value)

	return nil
}

func (l *Lexer) exportHandler(idx int, params []string) error {
	if len(params) > 1 {
		return errors.New("ERROR: export can't have more than one param")
	}

	if len(params) < 1 {
		return errors.New("ERROR: export can't have less than one param")
	}

	// check the param format
	// the param format needs to be a valid path
	path := params[0]

	path = strings.TrimSpace(path)
	path = filepath.Clean(path)

	osPath, _ := os.Getwd()
	path = filepath.Join(osPath, path)

	if !fileIsVideo(path) {
		return errors.New("ERROR: provided filepath is not a valid video type")
	}

	key := createKey(EXPORT, l.Line)
	value := Instruction{
		Params: []string{path},
		Order:  uint(idx),
	}
	l.addToken(key, value)

	return nil
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

	lexer := NewLexer()

	lexer.Tokenize(content)

	type OrderedInstruction struct {
		Key   string
		Value Instruction
	}

	var ordered []OrderedInstruction
	for k, v := range lexer.Tokens {
		ordered = append(ordered, OrderedInstruction{k, v})
	}

	sort.Slice(ordered, func(i, j int) bool {
		return ordered[i].Value.Order < ordered[j].Value.Order
	})

	for _, item := range ordered {
		fmt.Println(item.Key, item.Value)
	}
}
