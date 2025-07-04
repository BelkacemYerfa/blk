package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

type TOKEN = string

const (
	PUSH           TOKEN = "push"
	TRIM           TOKEN = "trim"
	EXPORT         TOKEN = "export"
	CONCAT         TOKEN = "concat"
	THUMBNAIL_FROM TOKEN = "thumbnail_from"
)

var (
	videoExts = []string{
		".mp4", ".mov", ".avi", ".mkv",
		".webm", ".flv", ".wmv",
	}
	imageExts = []string{
		".jpg", ".jpeg", ".png", ".gif",
		".bmp", ".webp", ".tiff",
	}
)

type Instruction struct {
	Command TOKEN
	Params  []string
	Order   uint
}

type Parser struct {
	AST  []Instruction
	Line uint
}

func NewParser() *Parser {
	return &Parser{
		AST:  make([]Instruction, 0),
		Line: 0,
	}
}

func (l *Parser) Tokenize(content string) {
	chunks := strings.Split(content, "\n")

	for idx, chunk := range chunks {
		list := strings.Split(strings.TrimSpace(chunk), " ")

		operation := list[0]
		params := list[1:]

		params = slices.DeleteFunc(params, func(param string) bool {
			return len(param) <= 0
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
		case THUMBNAIL_FROM:
			if err := l.thumbnailHandler(idx, params); err != nil {
				fmt.Printf(
					"%v\nLine: %v\nInstruction: %s", err, idx+1, chunk,
				)
				return
			}
		}
	}
}

func (l *Parser) addNode(value Instruction) {
	l.AST = append(l.AST, value)
}

func (l *Parser) videoPathCheck(path string) error {
	path = strings.TrimSpace(path)

	_, err := isValidPathFormat(path)

	if err != nil {
		return err
	}

	osPath, _ := os.Getwd()
	path = filepath.Join(osPath, path)
	path = filepath.Clean(path)

	if !checkFileIsOfTypeMode(path, VIDEO) {
		return errors.New("ERROR: file extension needs to be a video")
	}

	return nil
}

func (l *Parser) imagePathCheck(path string) error {
	path = strings.TrimSpace(path)

	_, err := isValidPathFormat(path)

	if err != nil {
		return err
	}

	osPath, _ := os.Getwd()
	path = filepath.Join(osPath, path)
	path = filepath.Clean(path)

	if !checkFileIsOfTypeMode(path, IMAGE) {
		return errors.New("ERROR: file extension needs to be a video")
	}

	return nil
}

func (l *Parser) pushHandler(idx int, params []string) error {
	if len(params) > 1 {
		return errors.New("ERROR: push can't have more than one param")
	}

	if len(params) < 1 {
		return errors.New("ERROR: push can't have less than one param")
	}

	// check the param format
	// the param format needs to be a valid path
	path := params[0]

	if err := l.videoPathCheck(path); err != nil {
		return err
	}

	value := Instruction{
		Command: PUSH,
		Params:  []string{path},
		Order:   uint(idx),
	}
	l.addNode(value)

	return nil
}

func (l *Parser) trimHandler(idx int, params []string) error {
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
		if err := l.videoPathCheck(videoTarget); err != nil {
			return err
		}
	}

	// check the time format of both start and end

	for _, param := range params {
		if checkTimeTrimFormatValid(param) {
			return fmt.Errorf("ERROR: %v isn't in valid format to be used as time", param)
		}
	}

	trimParams := append([]string{}, params[:2]...)
	trimParams = append(trimParams, videoTarget)
	value := Instruction{
		Command: TRIM,
		Params:  trimParams,
		Order:   uint(idx),
	}
	l.addNode(value)
	return nil
}

func (l *Parser) concatHandler(idx int, params []string) error {
	if len(params) > 0 {
		return errors.New("ERROR: concat doesn't have params")
	}

	value := Instruction{
		Command: CONCAT,
		Params:  []string{},
		Order:   uint(idx),
	}

	l.addNode(value)

	return nil
}

func (l *Parser) thumbnailHandler(idx int, params []string) error {
	if len(params) > 2 {
		return errors.New("ERROR: thumbnail_from can't have more than two params")
	}

	if len(params) < 2 {
		return errors.New("ERROR: thumbnail_from can't have less than two params")
	}

	format := params[0]

	// check if it follows the time format
	timeFormat := `^\d{2}:\d{2}:\d{2}$`
	if matched, _ := regexp.MatchString(timeFormat, format); matched {
		if checkTimeTrimFormatValid(format) {
			return fmt.Errorf("ERROR: %v isn't in valid format to be used as time", format)
		}
	} else {
		// check if it is a frame number
		_, err := strconv.Atoi(format)

		if err != nil {
			return fmt.Errorf("invalid number format, %v", err)
		}
	}

	output := params[1]

	// this may return an error cause it forces to use a video format only
	if err := l.imagePathCheck(output); err != nil {
		return err
	}

	value := Instruction{
		Command: THUMBNAIL_FROM,
		Params:  []string{format, output},
		Order:   uint(idx),
	}

	l.addNode(value)

	return nil
}

func (l *Parser) exportHandler(idx int, params []string) error {
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

	_, err := isValidPathFormat(path)

	if err != nil {
		return err
	}

	osPath, _ := os.Getwd()
	path = filepath.Join(osPath, path)

	if !checkFileIsOfTypeMode(path, VIDEO) {
		return errors.New("ERROR: file extension needs to be a video")
	}

	value := Instruction{
		Command: EXPORT,
		Params:  []string{path},
		Order:   uint(idx),
	}
	l.addNode(value)

	return nil
}

func isValidPathFormat(path string) (bool, error) {
	if strings.ContainsAny(path, `<>:"|?*`) {
		return false, errors.New("ERROR: special characters like (<>:'|?*) are not allowed")
	}

	// Must be valid filepath format
	if !filepath.IsLocal(path) {
		return false, errors.New("ERROR: path is invalid")
	}

	// Must have extension
	ext := filepath.Ext(path)
	if ext == "" {
		return false, errors.New("ERROR: file at the end of path needs to have an extension")
	}
	return true, nil
}

type Mode = string

const (
	VIDEO Mode = "video"
	IMAGE Mode = "image"
)

func checkFileIsOfTypeMode(path string, mode Mode) bool {
	ext := filepath.Ext(path)

	modeOptions := make([]string, 0)
	switch mode {
	case VIDEO:
		modeOptions = videoExts
	case IMAGE:
		modeOptions = imageExts
	}

	return checkIfElementExist(modeOptions, ext)
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

	Parser := NewParser()

	Parser.Tokenize(content)

	for _, v := range Parser.AST {
		fmt.Println(v)
	}
}
