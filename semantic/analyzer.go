package semantic

import "go/ast"

type Analyzer struct {
	errors   []error
	symtab   *SymbolTable
	types    *TypeChecker
	filename string
}

func NewAnalyzer(filename string) *Analyzer {
	return &Analyzer{
		errors:   make([]error, 0),
		symtab:   NewSymTable(),
		types:    NewTypeChecker(),
		filename: filename,
	}
}

func (*Analyzer) Analyze(node ast.Node) error {
	return nil
}
