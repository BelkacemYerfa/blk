package semantic

import (
	"blk/ast"
	"fmt"
)

type Analyzer struct {
	filename string
	symtab   *SymbolTable
	types    *TypeChecker
	errors   *ErrorCollector
}

func NewAnalyzer(filename string) *Analyzer {
	symtab := NewSymTable()
	errCollector := NewErrorCollector(filename)
	return &Analyzer{
		errors:   errCollector,
		symtab:   symtab,
		types:    NewTypeChecker(filename, symtab, errCollector),
		filename: filename,
	}
}

func (a *Analyzer) GetErrors() []error {
	return a.errors.errors
}

func (a *Analyzer) Analyze(node *ast.Program) {
	// start collecting
	a.collectSymbols(node)

	a.types.check(node)

	if main := a.symtab.CurrentScope.Resolve("main"); main == nil {
		a.errors.error(ERROR, node.GetToken(), fmt.Errorf("no entry point found (main function), an entry point is required"))
	}

	for _, sym := range a.symtab.CurrentScope.Symbols {
		if sym.DeclNode != nil && !sym.Used {
			a.errors.error(WARNING, sym.DeclNode.GetToken(), fmt.Errorf("%v is not used", sym.Name))
		}
	}

}

func (a *Analyzer) collectSymbols(node ast.Node) {

	switch n := node.(type) {
	case *ast.Program:
		for _, stmt := range n.Statements {
			a.collectSymbols(stmt)
		}

	case *ast.Declaration:
		a.collectDeclSymbol(n)

	case *ast.TypeDeclaration:
		a.collectTypeSymbol(n)

	case *ast.ScopeStatement:
		for _, stmt := range n.Body.Body {
			a.collectSymbols(stmt)
		}
	}
}

func (a *Analyzer) collectExpressionSymbol(node ast.Expression) {

	switch n := node.(type) {
	case *ast.FunctionExpression:
		a.symtab.EnterScope()
		defer a.symtab.ExitScope()

		if n.Self.Value != "" {
			// error out, for now, later check for the struct
			errMsg := fmt.Sprintf("self keyword can only be used withing struct context, but %v function is not a method, so either consider removing it or make this method a part of some struct", a.errors.highlight(n.Token.LiteralToken.Text, Yellow))
			a.errors.error(ERROR, n.Self.Token, errMsg)
			return
		}

		for _, arg := range n.Args {
			if arg.DefaultValue != nil {
				inferred := a.types.inferExpr(arg.DefaultValue)

				if !a.types.typesCompatible(arg.Type, inferred) {
					errMsg := fmt.Sprintf("type mismatch on %v argument, explicit type %v doesn't match the inferred type %v, change the explicit type or the associated value", a.errors.highlight(arg.Name.Value, Yellow), a.errors.highlight(arg.Type, Red), a.errors.highlight(inferred, Green))
					a.errors.error(ERROR, arg.Token, errMsg)
					return
				}
			}

			if err := a.symtab.CurrentScope.Define(arg.Name.Value, &Symbol{
				Name:      arg.Name.Value,
				Kind:      arg.Type,
				IsMutable: true,
				DeclNode:  arg.Name,
			}); err != nil {
				a.errors.error(ERROR, arg.Token, err)
				return
			}

		}

		for _, stmt := range n.Body.Body {
			a.collectSymbols(stmt)

			if s, ok := stmt.(*ast.Declaration); ok {
				if len(s.Value) > 0 {
					a.types.inferExpr(s.Value[0])
				}
			}
		}

		a.types.checkFunctionBody(n)

		for _, sym := range a.symtab.CurrentScope.Symbols {
			if sym.DeclNode != nil && !sym.Used {
				a.errors.error(WARNING, sym.DeclNode.GetToken(), fmt.Errorf("%v is not used", sym.Name))
			}
		}

	case *ast.BlockExpression:
		a.symtab.EnterScope()
		defer a.symtab.ExitScope()

		for _, stmt := range n.Body {
			a.collectSymbols(stmt)
		}

		for _, sym := range a.symtab.CurrentScope.Symbols {
			if sym.DeclNode != nil && !sym.Used {
				a.errors.error(WARNING, sym.DeclNode.GetToken(), fmt.Errorf("%v is not used", sym.Name))
			}
		}

	}
}

func (a *Analyzer) collectDeclSymbol(node *ast.Declaration) {
	var declarationType ast.Type

	if node.Type != nil {
		// explicit type
		declarationType = a.types.unalias(node.Type)
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
		a.errors.error(ERROR, node.Token, err)
		return
	}

	// body check of different expression such as functions, if blocks, switches, ...ect
	if len(node.Value) > 0 {
		a.collectExpressionSymbol(node.Value[0])
	}
}

func (a *Analyzer) collectTypeSymbol(node *ast.TypeDeclaration) {
	name := node.Alias.String()
	declarationType := a.types.unalias(node.Type)

	if err := a.symtab.CurrentScope.Define(name, &Symbol{
		Name:     node.Alias.String(),
		Kind:     declarationType,
		DeclNode: node,
	}); err != nil {
		a.errors.error(ERROR, node.Token, err)
		return
	}
}
