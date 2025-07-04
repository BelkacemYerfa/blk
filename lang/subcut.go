package main

import (
	"errors"
	"fmt"
	"os"
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
	_, err := time.Parse("hh:mm:ss", tm)
	if err != nil {
		return false
	}
	return true
}
