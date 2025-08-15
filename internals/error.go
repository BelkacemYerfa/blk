package internals

import (
	"blk/lexer"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
)

// This file handles an error collector obj

type ErrorCollector struct {
	Tokens []lexer.Token
	Errors []error
}

func NewErrorCollector(tokens []lexer.Token) *ErrorCollector {
	return &ErrorCollector{
		Tokens: tokens,
		Errors: make([]error, 0),
	}
}

func (ec *ErrorCollector) Add(errMsg error) {
	_, found := slices.BinarySearchFunc(ec.Errors, errMsg, func(a, b error) int {
		return strings.Compare(a.Error(), b.Error())
	})
	if !found {
		ec.Errors = append(ec.Errors, errMsg)
	}
}

func (ec *ErrorCollector) Error(tok lexer.Token, msg string) error {
	errMsg := fmt.Sprintf("\033[1;90m%s:%d:%d:\033[0m\n\n", "main.blk", tok.Row, tok.Col)

	// Build row set map
	rowSet := make(map[int][]lexer.Token)
	for _, t := range ec.Tokens {
		rowSet[t.Row] = append(rowSet[t.Row], t)
	}

	// Collect sorted rows
	rows := []int{}
	for row := range rowSet {
		rows = append(rows, row)
	}
	sort.Ints(rows)

	// Find closest previous and next row
	var prevRow, nextRow int
	prevRow, nextRow = -1, -1
	for _, row := range rows {
		if row < tok.Row {
			prevRow = row
		} else if row > tok.Row && nextRow == -1 {
			nextRow = row
		}
	}

	// Build rowMap with only prevRow, tok.Row, nextRow
	rowMap := make(map[int][]lexer.Token)
	if prevRow != -1 {
		rowMap[prevRow] = rowSet[prevRow]
	}
	rowMap[tok.Row] = rowSet[tok.Row]
	if nextRow != -1 {
		rowMap[nextRow] = rowSet[nextRow]
	}

	// Format rows
	formattedRows := []int{}
	for row := range rowMap {
		formattedRows = append(formattedRows, row)
	}
	sort.Ints(formattedRows)

	for _, row := range formattedRows {
		currentLine := rowMap[row]
		lineContent := ""
		lastCol := 0

		for _, t := range currentLine {
			if t.Col > lastCol {
				lineContent += strings.Repeat(" ", t.Col-lastCol)
			}
			if t.Kind == lexer.TokenString {
				t.Text = fmt.Sprintf(`"%s"`, t.Text)
			}
			lineContent += t.Text
			lastCol = t.Col + len(t.Text)
		}

		lineNumStr := fmt.Sprintf("%d", row)
		errMsg += fmt.Sprintf("%s    %s\n", lineNumStr, lineContent)

		if row == tok.Row {
			spacesBeforeLineNum := len(lineNumStr)
			spacesAfterLineNum := 4
			spacesBeforeToken := tok.Col

			totalSpaces := spacesBeforeLineNum + spacesAfterLineNum + spacesBeforeToken

			errorIndicator := strings.Repeat(" ", totalSpaces)
			errMsg += errorIndicator + "\033[1;31m"
			repeat := len(tok.Text)
			if repeat == 0 {
				repeat = 1
			}
			errMsg += strings.Repeat("^", repeat)
			errMsg += "\033[0m\n"
		}
	}

	errMsg += msg
	return errors.New(errMsg)
}

func (ec *ErrorCollector) GetErrors() []error {
	return ec.Errors
}
