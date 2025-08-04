package repl

import (
	"blk/interpreter"
	"blk/lexer"
	"blk/parser"
	"bufio"
	"fmt"
	"io"
)

const PROMPT = `>>>`

func Start(in io.Reader, out io.Writer) {
	scanner := bufio.NewScanner(in)
	for {
		fmt.Print(PROMPT)
		scanned := scanner.Scan()
		if !scanned {
			return
		}
		line := scanner.Text()
		l := lexer.NewLexer("", line)
		p := parser.NewParser(l.Tokenize(), "")
		program := p.Parse()
		if len(p.Errors) != 0 {
			for _, err := range p.Errors {
				fmt.Println(err)
			}
			continue
		}
		evaluated := interpreter.Eval(program)
		if evaluated != nil {
			io.WriteString(out, evaluated.Inspect())
			io.WriteString(out, "\n")
		}
	}
}
