package macro

import (
	"blk/lexer"
	"errors"
	"fmt"
	"slices"
	"strings"
)

type ErrorCollector struct {
	filePath string
	errors   []error
}

func NewErrorCollector(filePath string) *ErrorCollector {
	return &ErrorCollector{
		filePath: filePath,
		errors:   make([]error, 0),
	}
}

type HighlightFlag string

const (
	Reset   HighlightFlag = "\033[0m"
	Red     HighlightFlag = "\033[31m"
	Green   HighlightFlag = "\033[32m"
	Yellow  HighlightFlag = "\033[33m"
	Blue    HighlightFlag = "\033[34m"
	Magenta HighlightFlag = "\033[35m"
	Cyan    HighlightFlag = "\033[36m"
	Gray    HighlightFlag = "\033[37m"
	White   HighlightFlag = "\033[97m"
)

func (ec *ErrorCollector) highlight(msg any, flag HighlightFlag) string {
	nwMsg := ""

	switch flag {
	case Red:
		nwMsg = fmt.Sprintf("%v%v%v", Red, msg, Reset)
	case Green:
		nwMsg = fmt.Sprintf("%v%v%v", Green, msg, Reset)

	case Yellow:
		nwMsg = fmt.Sprintf("%v%v%v", Yellow, msg, Reset)

	case Blue:
		nwMsg = fmt.Sprintf("%v%v%v", Blue, msg, Reset)

	case Magenta:
		nwMsg = fmt.Sprintf("%v%v%v", Magenta, msg, Reset)

	case Cyan:
		nwMsg = fmt.Sprintf("%v%v%v", Cyan, msg, Reset)

	case Gray:
		nwMsg = fmt.Sprintf("%v%v%v", Gray, msg, Reset)

	case White:
		nwMsg = fmt.Sprintf("%v%v%v", White, msg, Reset)
	}

	return nwMsg
}

type Level int

const (
	zeroed Level = iota
	WARNING
	ERROR
)

func (ec *ErrorCollector) add(err error) {
	if _, found := slices.BinarySearchFunc(ec.errors, err, func(a, b error) int {
		return strings.Compare(a.Error(), b.Error())
	}); !found {
		ec.errors = append(ec.errors, err)
	}
}

func (ec *ErrorCollector) error(level Level, tok lexer.Token, msg ...interface{}) error {
	lvl := "ERROR"
	if level == WARNING {
		lvl = "WARNING"
	}
	errMsg := fmt.Sprintf("\033[1;90m%s:%d:%d:\033[0m %s: %s", ec.filePath, tok.Row, tok.Col, lvl, fmt.Sprint(msg...))
	err := errors.New(errMsg)
	ec.add(err)
	return err
}
