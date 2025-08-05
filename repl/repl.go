package repl

import (
	"blk/interpreter"
	"blk/lexer"
	"blk/object"
	"blk/parser"
	"bufio"
	"fmt"
	"io"
	"strings"
)

const PROMPT = `>>>`

func Start(in io.Reader, out io.Writer) {
	scanner := bufio.NewScanner(in)
	env := object.NewEnvironment(nil)

	for {
		fmt.Print(PROMPT)
		scanned := scanner.Scan()
		if !scanned {
			return
		}
		line := scanner.Text()
		if strings.Contains("exit()", line) {
			break
		}
		l := lexer.NewLexer("", line)
		p := parser.NewParser(l.Tokenize(), "")
		program := p.Parse()
		if len(p.Errors) != 0 {
			for _, err := range p.Errors {
				fmt.Println(err)
			}
			continue
		}
		i := interpreter.NewInterpreter(env)
		evaluated := i.Eval(program)
		if evaluated != nil {
			io.WriteString(out, evaluated.Inspect())
			io.WriteString(out, "\n")
		}
	}
}
