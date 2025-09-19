package semantic

import (
	"blk/ast"
)

type Analyzer struct {
	errors   []error
	symtab   *SymbolTable
	types    *TypeChecker
	filename string
}

func NewAnalyzer(filename string) *Analyzer {
	symtab := NewSymTable()
	return &Analyzer{
		errors:   make([]error, 0),
		symtab:   symtab,
		types:    NewTypeChecker(filename, symtab),
		filename: filename,
	}
}

func (a *Analyzer) GetErrors() []error {
	return a.errors
}

func (a *Analyzer) add(err error) {
	if len(err.Error()) > 0 {
		a.errors = append(a.errors, err)
	}
}

func (a *Analyzer) Analyze(node *ast.Program) {
	// start collecting
	if err := a.collectSymbols(node); err != nil {
		a.add(err)
	}

	a.types.check(node)
	a.errors = append(a.errors, a.types.errors...)
}

func (a *Analyzer) collectSymbols(node ast.Node) error {

	switch n := node.(type) {
	case *ast.Program:
		for _, stmt := range n.Statements {
			if err := a.collectSymbols(stmt); err != nil {
				return err
			}
		}

	case *ast.Declaration:
		return a.collectDeclSymbol(n)

	case *ast.ScopeStatement:
		for _, stmt := range n.Body.Body {
			if err := a.collectSymbols(stmt); err != nil {
				return err
			}
		}

	case *ast.UsingStatement:
		return a.collectUsingSymbol(n)
	}

	return nil
}

func (a *Analyzer) collectUsingSymbol(node *ast.UsingStatement) error {
	declarationType := a.types.inferExpr(node.Alias)
	return a.symtab.CurrentScope.Define(node.String(), &Symbol{
		Name:     node.String(),
		Kind:     declarationType,
		DeclNode: node,
	})
}

func (a *Analyzer) collectDeclSymbol(node *ast.Declaration) error {

	var declarationType ast.Type

	if node.Type != nil {
		// explicit type
		declarationType = node.Type
	} else {
		// first support only first value
		declarationType = a.types.inferExpr(node.Value[0])
	}

	// better to have errors returned later
	err := a.symtab.CurrentScope.Define(node.Name[0].String(), &Symbol{
		Name:      node.Name[0].String(),
		Kind:      declarationType,
		IsMutable: node.Mutable,
		DeclNode:  node,
	})

	// body check of stuff here

	return err
}
