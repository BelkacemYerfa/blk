package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
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

var (
	stack []string
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
	return idx < len(videoExts) && videoExts[idx] == element
}

func fileIsVideo(path string) bool {
	ext := filepath.Ext(path)

	return checkIfElementExist(videoExts, ext)
}

func pushHandler(params []string) error {
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

	if checkIfElementExist(stack, path) {
		// this means it is already in the stack no need to push it again
		return nil
	}

	if !fileIsVideo(path) {
		return errors.New("ERROR: provided filepath is not a valid video type")
	}

	stack = append(stack, path)
	return nil
}

func trimHandler(params []string) error {
	return nil
}

func concatHandler(params []string) error {
	return nil
}

func exportHandler(params []string) error {

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

	chunks := strings.Split(content, "\n")

	for idx, chunk := range chunks {
		list := strings.Split(chunk, " ")

		operation := list[0]
		params := list[1:]

		params = slices.DeleteFunc(params, func(param string) bool {
			return len(strings.TrimSpace(param)) == 0
		})

		switch strings.ToLower(operation) {
		case PUSH:
			if err := pushHandler(params); err != nil {
				fmt.Printf(
					"%v\nLine: %v\nInstruction: %s", err, idx+1, chunk,
				)
				return
			}
		case TRIM:
		case CONCAT:
		case EXPORT:
			if err := exportHandler(params); err != nil {
				fmt.Printf(
					"%v\nLine: %v\nInstruction: %s", err, idx+1, chunk,
				)
				return
			}

		}
	}

	fmt.Println(stack)
}
