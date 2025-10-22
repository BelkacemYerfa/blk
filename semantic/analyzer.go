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
	a.passCollectSymbols(node)

	if main := a.symtab.CurrentScope.Resolve("main"); main == nil {
		a.errors.error(ERROR, node.GetToken(), fmt.Errorf("no entry point found (main function), an entry point is required"))
	}

	a.passTypeResolution(node)

	a.passTypeChecking(node)
	a.secondPassCheck()

	for _, sym := range a.symtab.CurrentScope.Symbols {
		if sym.DeclNode != nil && !sym.Used && sym.Name != "_" {
			a.errors.error(WARNING, sym.DeclNode.GetToken(), fmt.Errorf("%v is not used", sym.Name))
		}
	}
}

func (a *Analyzer) passCollectSymbols(node ast.Node) {
	switch n := node.(type) {
	case *ast.Program:
		for _, stmt := range n.Statements {
			a.passCollectSymbols(stmt)
		}

	case *ast.Declaration:
		a.collectDeclSymbol(n)

	case *ast.TypeDeclaration:
		a.collectTypeSymbol(n)

	case *ast.ScopeStatement:
		for _, stmt := range n.Body.Body {
			a.passCollectSymbols(stmt)
		}
	}
}

func (a *Analyzer) passTypeResolution(node ast.Node) {
	switch n := node.(type) {
	case *ast.Program:
		for _, stmt := range n.Statements {
			a.passTypeResolution(stmt)
		}

	case *ast.TypeDeclaration:
		a.checkTypeExpression(n.Type)

	case *ast.Declaration:
		name := n.Name[0].String()
		sym := a.symtab.CurrentScope.Resolve(name)

		if sym.Kind == nil && n.Value != nil {
			// no explicit type, infer from value
			sym.Kind = a.types.inferExpr(n.Value[0])
			nwSym := *sym
			a.types.symtab.CurrentScope.Update(name, &nwSym)
		}

	case *ast.ScopeStatement:
		a.symtab.EnterScope()
		defer a.symtab.ExitScope()
		for _, stmt := range n.Body.Body {
			a.passTypeResolution(stmt)
		}
	}
}

func (a *Analyzer) passTypeChecking(node *ast.Program) {
	a.types.check(node)
}

func (a *Analyzer) secondPassCheck() {
	for _, sym := range a.symtab.CurrentScope.Symbols {
		switch dclN := sym.DeclNode.(type) {
		case *ast.Declaration:
			a.checkDeclarationExpression(dclN.Value)
		}
	}
}

func (a *Analyzer) checkDeclarationExpression(nodes []ast.Expression) {
	for _, node := range nodes {
		switch n := node.(type) {
		case *ast.FunctionExpression:
			a.checkFunctionExpression(n)

		case *ast.BlockExpression:
			a.symtab.EnterScope()
			defer a.symtab.ExitScope()

			for _, stmt := range n.Body {
				a.passCollectSymbols(stmt)
			}

			for _, sym := range a.symtab.CurrentScope.Symbols {
				if sym.DeclNode != nil && !sym.Used {
					a.errors.error(WARNING, sym.DeclNode.GetToken(), fmt.Errorf("%v is not used", sym.Name))
				}
			}
		}
	}
}

func (a *Analyzer) checkFunctionExpression(fn *ast.FunctionExpression) {
	a.symtab.EnterScope()
	defer a.symtab.ExitScope()

	for _, arg := range fn.Args {
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

	for _, stmt := range fn.Body.Body {
		a.passCollectSymbols(stmt)

		if s, ok := stmt.(*ast.Declaration); ok {
			if len(s.Value) > 0 {
				a.types.inferExpr(s.Value[0])
			}
		}
	}

	a.types.checkFunctionBody(fn)

	for _, sym := range a.symtab.CurrentScope.Symbols {
		if sym.DeclNode != nil && !sym.Used {
			a.errors.error(WARNING, sym.DeclNode.GetToken(), fmt.Errorf("%v is not used", sym.Name))
		}
	}
}

func (a *Analyzer) checkTypeExpression(node ast.Type) {
	switch n := node.(type) {
	case *ast.EnumType:
		a.checkEnumType(n)
	case *ast.UnionType:
		a.checkUnionType(n)
	case *ast.StructType:
		a.checkStructType(n)
	case *ast.AliasType:
		a.checkAliasType(n)
	}
}

func (a *Analyzer) checkEnumType(n *ast.EnumType) int {
	startIdx := 0
	prevIdxValue := startIdx

	for _, expr := range n.Body {
		// check that right side is of type int
		if len(expr.Right) > 0 {
			rhs := expr.Right[0]
			infType := a.types.inferExpr(rhs)

			if infType.Type() < ast.TypeSInt8 || infType.Type() > ast.TypeUInt64 {
				a.errors.error(WARNING, rhs.GetToken(), fmt.Errorf("assigned value for enums needs to be of type int only, instead we got %v", a.errors.highlight(infType, Red)))
				return 0
			}

			rhsValue := int(rhs.(*ast.IntegerLiteral).Value)
			if startIdx == 0 {
				startIdx = rhsValue
			}

			if rhsValue <= prevIdxValue {
				a.errors.error(WARNING, rhs.GetToken(), fmt.Errorf("%v value is smaller or equal to the previous value (%v), consider changing it, or remove the associated value and the compiler will associate the correct next value", a.errors.highlight(rhsValue, Red), a.errors.highlight(prevIdxValue, Yellow)))
				return 0
			}

			prevIdxValue = rhsValue
			startIdx++

		} else {
			expr.Right = append(expr.Right, &ast.IntegerLiteral{
				Token: expr.Token,
				Value: int64(startIdx),
			})
		}
	}
	return startIdx
}

func (a *Analyzer) checkUnionType(n *ast.UnionType) {
	// check that there is no duplicates
	for i, cnType := range n.Body {
		for _, onType := range n.Body[i+1:] {
			if onType.Type() == cnType.Type() {
				a.errors.error(ERROR, n.Token, fmt.Sprintf("%v type is already in the union, duplicates are not allowed", a.errors.highlight(onType, Red)))
			}
		}
	}
}

func (a *Analyzer) checkAliasType(n *ast.AliasType) {
	if _, err := Unalias(a.errors, a.symtab, n); err != nil {
		return
	}
}

func (a *Analyzer) checkStructType(n *ast.StructType) {
	for _, field := range n.Fields {
		a.checkTypeExpression(field.Type)
		// check with associated default value if	 exists
		if len(field.Value) > 0 {
			associatedValue := field.Value[0]
			// associated value can't be a identifier here only a literal
			if _, ok := associatedValue.(ast.Literal); !ok {
				a.errors.error(ERROR, field.Token, "associated value of the struct type can't be an identifier, it can only be a literal type (int, float, string, char, ...)")
				return
			}
			inferred := a.types.inferExpr(associatedValue)
			name := field.Name[0]
			if !a.types.typesCompatible(field.Type, inferred) {
				errMsg := fmt.Sprintf("type mismatch on %v argument, explicit type %v doesn't match the inferred type %v, change the explicit type or the associated value", a.errors.highlight(name, Yellow), a.errors.highlight(field.Type, Red), a.errors.highlight(inferred, Green))
				a.errors.error(ERROR, field.Token, errMsg)
				return
			}
		}
	}
}

func (a *Analyzer) collectDeclSymbol(node *ast.Declaration) {
	var declarationType ast.Type

	if node.Type != nil {
		// explicit type
		unaliasType, err := Unalias(a.errors, a.symtab, node.Type)
		if err != nil {
			a.errors.add(err)
			return
		}
		declarationType = unaliasType
	} else {
		// // first support only first value
		// declarationType = a.types.inferExpr(node.Value[0])

		// // directives
		// if _, ok := node.Value[0].(*ast.FunctionExpression); !ok {
		// 	if node.Inline {
		// 		a.errors.error(ERROR, node.GetToken(), fmt.Errorf("#inline directive can only be used with function expressions"))
		// 	}

		// 	dc, ok := node.Directive["#init"]

		// 	if ok {
		// 		a.errors.error(ERROR, dc.GetToken(), fmt.Errorf("#init directive can only be used with function expressions"))
		// 	}

		// 	dc, ok = node.Directive["#fini"]

		// 	if ok {
		// 		a.errors.error(ERROR, dc.GetToken(), fmt.Errorf("#fini directive can only be used with function expressions"))
		// 	}
		// } else {
		// 	_, initExists := node.Directive["#init"]
		// 	dc, finiExists := node.Directive["#fini"]

		// 	if initExists && finiExists {
		// 		a.errors.error(ERROR, dc.GetToken(), fmt.Errorf("can't use the both %v and %v directives on function expression, one only could exist", a.errors.highlight("#fini", Yellow), a.errors.highlight("#init", Yellow)))
		// 	}

		// 	// get the function signature
		// 	fnType := (declarationType).(*ast.FunctionType)

		// 	if initExists || finiExists {
		// 		if len(fnType.Args) > 0 {
		// 			a.errors.error(ERROR, fnType.Token, fmt.Errorf("%v function has #init or #fini directive, thus can't have arguments", a.errors.highlight(node.Name[0], Yellow)))
		// 		}

		// 		if len(fnType.Return) > 0 {
		// 			if len(fnType.Return) >= 1 && fnType.Return[0].Type() != ast.TypeVoid {
		// 				a.errors.error(ERROR, fnType.Token, fmt.Errorf("%v function has #init or #fini directive, thus can't have returned values", a.errors.highlight(node.Name[0], Yellow)))
		// 			}
		// 		}
		// 	}

		// 	inlineExists := node.Inline
		// 	dc, noInlineExists := node.Directive["#no_inline"]

		// 	if inlineExists && noInlineExists {
		// 		a.errors.error(ERROR, dc.GetToken(), fmt.Errorf("can't use the both %v and %v directives on function expression, one only could exist", a.errors.highlight("#inline", Yellow), a.errors.highlight("#no_inline", Yellow)))
		// 	}
		// }
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
}

func (a *Analyzer) collectTypeSymbol(node *ast.TypeDeclaration) {
	name := node.Alias.String()

	if err := a.symtab.CurrentScope.Define(name, &Symbol{
		Name:     node.Alias.String(),
		Kind:     node.Type,
		DeclNode: node,
	}); err != nil {
		a.errors.error(ERROR, node.Token, err)
		return
	}
}
