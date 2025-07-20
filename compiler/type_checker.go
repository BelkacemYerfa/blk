package compiler

import (
	"blk/parser"
	"fmt"
)

type TypeChecker struct {
	Symbols *SymbolTable
	Errors  []error
}

func NewTypeChecker() *TypeChecker {
	return &TypeChecker{
		Symbols: NewSymbolTable(),
	}
}

func (tc *TypeChecker) StmtChecker(ast *parser.Program) {
	for _, node := range ast.Statements {

		switch node.(type) {
		case *parser.LetStatement:
			fmt.Println(node)
		default:
		}

	}
}

func (tc *TypeChecker) visiteLetStatement(node *parser.LetStatement) {

}
