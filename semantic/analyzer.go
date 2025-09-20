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

func (a *Analyzer) collectExpressionSymbol(node ast.Expression) error {

	switch n := node.(type) {
	case *ast.FunctionExpression:
		a.symtab.EnterScope()
		defer a.symtab.ExitScope()

		for _, arg := range n.Args {
			if err := a.symtab.CurrentScope.Define(arg.Name.Value, &Symbol{
				Name:      arg.Name.Value,
				Kind:      arg.Type,
				IsMutable: true,
				DeclNode:  arg.Name,
			}); err != nil {
				return err
			}
		}

		for _, stmt := range n.Body.Body {
			if err := a.collectSymbols(stmt); err != nil {
				return err
			}

			if s, ok := stmt.(*ast.Declaration); ok {
				if len(s.Value) > 0 {
					a.types.inferExpr(s.Value[0])
				}
			}
		}

		a.types.checkFunctionBody(n)

	case *ast.BlockExpression:
		a.symtab.EnterScope()
		defer a.symtab.ExitScope()

		for _, stmt := range n.Body {
			if err := a.collectSymbols(stmt); err != nil {
				return err
			}
		}
	}

	return nil
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
	if err := a.symtab.CurrentScope.Define(node.Name[0].String(), &Symbol{
		Name:      node.Name[0].String(),
		Kind:      declarationType,
		IsMutable: node.Mutable,
		DeclNode:  node,
	}); err != nil {
		return err
	}

	// body check of different expression such as functions, if blocks, switches, ...ect
	if len(node.Value) > 0 {
		return a.collectExpressionSymbol(node.Value[0])
	}
	return nil
}
