package macro

import (
	"blk/ast"
	"blk/lexer"
	smt "blk/semantic"
	"fmt"
)

type MacroEngine struct {
	symtab *smt.SymbolTable
	errors *ErrorCollector
}

func NewMacroEngine(filePath string) *MacroEngine {
	return &MacroEngine{
		symtab: smt.NewSymTable(),
		errors: NewErrorCollector(filePath),
	}
}

func (a *MacroEngine) GetErrors() []error {
	return a.errors.errors
}

func (me *MacroEngine) MacroPass(node *ast.Program) {
	// start collecting
	me.collectSymbols(node)
	me.secondPassCheck()
}

func (me *MacroEngine) collectSymbols(node ast.Node) {
	switch n := node.(type) {
	case *ast.Program:
		for _, stmt := range n.Statements {
			me.collectSymbols(stmt)
		}

	case *ast.TypeDeclaration:
		me.collectTypeSymbol(n)
	}
}

func (me *MacroEngine) collectTypeSymbol(node *ast.TypeDeclaration) {
	name := node.Alias.String()
	declarationType, err := smt.Unalias(me.symtab, node.Type)

	if err != nil {
		me.errors.add(err)
		return
	}

	if err := me.symtab.CurrentScope.Define(name, &smt.Symbol{
		Name:     node.Alias.String(),
		Kind:     declarationType,
		DeclNode: node,
	}); err != nil {
		me.errors.error(ERROR, node.Token, err)
		return
	}
}

func (me *MacroEngine) secondPassCheck() {
	for _, sym := range me.symtab.CurrentScope.Symbols {
		switch dclN := sym.DeclNode.(type) {
		case *ast.TypeDeclaration:
			me.checkTypeExpression(dclN)
		}
	}
}

func (me *MacroEngine) checkTypeExpression(node *ast.TypeDeclaration) {
	switch n := node.Type.(type) {
	case *ast.EnumType:
		visited := make(map[string]bool)
		visited[node.Alias.Value] = true
		me.expandEnumMacros(node, n, visited)
	case *ast.StructType:
		me.expandStructMacros(n)
	}
}

// expandEnumMacros recursively expands all #bake directives in an enum
func (me *MacroEngine) expandEnumMacros(decl *ast.TypeDeclaration, enum *ast.EnumType, visited map[string]bool) error {
	var expandedBody []*ast.AssignExpression

	for _, expr := range enum.Body {
		// skip expressions without #bake directive
		if _, hasBake := expr.Directive[lexer.TokenBake]; !hasBake {
			expandedBody = append(expandedBody, expr)
			continue
		}

		// process each #bake target
		for _, embed := range expr.Embeddable {
			baked, err := me.expandBakeDirective(decl, embed, visited)
			if err != nil {
				me.errors.error(ERROR, embed.Token, err.Error())
				continue
			}

			// check for name conflicts before merging
			for _, bakedExpr := range baked {
				if me.hasConflict(expandedBody, bakedExpr) {
					me.errors.error(ERROR, embed.Token,
						fmt.Sprintf("%v already exists, consider renaming it",
							me.errors.highlight(bakedExpr.Left[0], Red)))
					continue
				}
				expandedBody = append(expandedBody, bakedExpr)
			}
		}
	}

	enum.Body = expandedBody
	return nil
}

// expandBakeDirective resolves and expands a single #bake target
func (me *MacroEngine) expandBakeDirective(
	parentDecl *ast.TypeDeclaration,
	embed *ast.Identifier,
	visited map[string]bool,
) ([]*ast.AssignExpression, error) {

	// resolve the baked enum
	sym := me.symtab.CurrentScope.Resolve(embed.Value)
	if sym == nil {
		return nil, fmt.Errorf("no declaration found with name %v",
			me.errors.highlight(embed.Value, Red))
	}

	enumType, ok := sym.Kind.(*ast.EnumType)
	if !ok {
		return nil, fmt.Errorf("#bake requires enum type, got %v",
			me.errors.highlight(sym.Kind, Red))
	}

	// detect cycles
	if visited[embed.Value] {
		return nil, fmt.Errorf("circular #bake detected between %v and %v",
			me.errors.highlight(parentDecl.Alias, Yellow),
			me.errors.highlight(embed.Value, Red))
	}

	// mark as visited and recursively expand
	visited[embed.Value] = true
	defer delete(visited, embed.Value) // Backtrack for other branches

	enumDecl, ok := sym.DeclNode.(*ast.TypeDeclaration)
	if !ok {
		return nil, fmt.Errorf("invalid declaration node for %v", embed.Value)
	}

	if err := me.expandEnumMacros(enumDecl, enumType, visited); err != nil {
		return nil, fmt.Errorf("failed to expand %v: %w", embed.Value, err)
	}

	// Return a copy of the expanded body to avoid shared mutations
	result := make([]*ast.AssignExpression, len(enumType.Body))
	copy(result, enumType.Body)

	return result, nil
}

// hasConflict checks if an enum variant name already exists
func (me *MacroEngine) hasConflict(existing []*ast.AssignExpression, candidate *ast.AssignExpression) bool {
	if len(candidate.Left) == 0 {
		return false
	}

	candidateName := candidate.Left[0].String()

	for _, expr := range existing {
		if len(expr.Left) > 0 && expr.Left[0].String() == candidateName {
			return true
		}
	}

	return false
}

func (me *MacroEngine) expandStructMacros(n *ast.StructType) {

}
