package semantic

import (
	"blk/ast"
	"blk/lexer"
	"fmt"
	"strings"
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
	a.secondPassCheck()

	if main := a.symtab.CurrentScope.Resolve("main"); main == nil {
		a.errors.error(ERROR, node.GetToken(), fmt.Errorf("no entry point found (main function), an entry point is required"))
	}

	for _, sym := range a.symtab.CurrentScope.Symbols {
		if sym.DeclNode != nil && !sym.Used && sym.Name != "_" {
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

func (a *Analyzer) secondPassCheck() {
	for _, sym := range a.symtab.CurrentScope.Symbols {
		switch dclN := sym.DeclNode.(type) {
		case *ast.TypeDeclaration:
			a.checkTypeExpression(dclN.Type)
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

func (a *Analyzer) checkTypeExpression(node ast.Type) {
	switch n := node.(type) {
	case *ast.EnumType:
		a.checkEnumType(n)
	case *ast.StructType:
		a.checkStructType(n)
	}
}

func (a *Analyzer) checkEnumType(n *ast.EnumType) int {
	startIdx := 0
	prevIdxValue := startIdx

	for _, expr := range n.Body {
		if _, ok := expr.Directive[lexer.TokenBake]; ok {
			// search for the identifier & bake all of the fields into the current enum

			for _, embed := range expr.Embeddable {
				sym := a.symtab.CurrentScope.Resolve(embed.Value)

				if sym == nil {
					(a.errors.error(ERROR, embed.Token, fmt.Sprintf("no declaration found with %v name, consider defining it", a.errors.highlight(embed.Value, Red))))
					return 0
				}

				if sym.Kind.Type() != ast.TypeEnum {
					(a.errors.error(ERROR, embed.Token, fmt.Sprintf("identifier used with bake directive needs to be of type enum, but we got %v as the type of %v, consider removing it", a.errors.highlight(sym.Kind, Red), a.errors.highlight(sym.Kind, Yellow))))
					return 0
				}

				embeddableEnum := sym.Kind.(*ast.EnumType)
				if lastIdx := a.checkEnumType(embeddableEnum); lastIdx != 0 {
					startIdx = lastIdx
				}
				// check if already an enum field of the same name exists or not, if yes error out
				for _, embedExpr := range embeddableEnum.Body {
					for _, expr := range n.Body {
						if len(expr.Left) > 0 && len(embedExpr.Left) > 0 {
							equal := strings.Contains(expr.Left[0].String(), embedExpr.Left[0].String()) || strings.Contains(embedExpr.Left[0].String(), expr.Left[0].String())
							if equal {
								(a.errors.error(ERROR, embed.Token, fmt.Sprintf("%v already exists in embeddable enum %v, consider renaming it or remove it", a.errors.highlight(expr.Left[0], Red), a.errors.highlight(embed.Value, Yellow))))
								return 0
							}
						}
					}
					n.Body = append(n.Body, embedExpr)
				}
			}

		} else {
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
	}
	return startIdx
}

func (a *Analyzer) checkStructType(n *ast.StructType) {

}

func (a *Analyzer) collectDeclSymbol(node *ast.Declaration) {
	var declarationType ast.Type

	if node.Type != nil {
		// explicit type
		declarationType = a.types.unalias(node.Type)
	} else {
		// first support only first value
		declarationType = a.types.inferExpr(node.Value[0])

		// directives
		if _, ok := node.Value[0].(*ast.FunctionExpression); !ok {
			if node.Inline {
				a.errors.error(ERROR, node.GetToken(), fmt.Errorf("#inline directive can only be used with function expressions"))
			}

			dc, ok := node.Directive["#init"]

			if ok {
				a.errors.error(ERROR, dc.GetToken(), fmt.Errorf("#init directive can only be used with function expressions"))
			}

			dc, ok = node.Directive["#fini"]

			if ok {
				a.errors.error(ERROR, dc.GetToken(), fmt.Errorf("#fini directive can only be used with function expressions"))
			}
		} else {
			_, initExists := node.Directive["#init"]
			dc, finiExists := node.Directive["#fini"]

			if initExists && finiExists {
				a.errors.error(ERROR, dc.GetToken(), fmt.Errorf("can't use the both %v and %v directives on function expression, one only could exist", a.errors.highlight("#fini", Yellow), a.errors.highlight("#init", Yellow)))
			}

			// get the function signature
			fnType := (declarationType).(*ast.FunctionType)

			if initExists || finiExists {
				if len(fnType.Args) > 0 {
					a.errors.error(ERROR, fnType.Token, fmt.Errorf("%v function has #init or #fini directive, thus can't have arguments", a.errors.highlight(node.Name[0], Yellow)))
				}

				if len(fnType.Return) > 0 {
					if len(fnType.Return) >= 1 && fnType.Return[0].Type() != ast.TypeVoid {
						a.errors.error(ERROR, fnType.Token, fmt.Errorf("%v function has #init or #fini directive, thus can't have returned values", a.errors.highlight(node.Name[0], Yellow)))
					}
				}
			}

			inlineExists := node.Inline
			dc, noInlineExists := node.Directive["#no_inline"]

			if inlineExists && noInlineExists {
				a.errors.error(ERROR, dc.GetToken(), fmt.Errorf("can't use the both %v and %v directives on function expression, one only could exist", a.errors.highlight("#inline", Yellow), a.errors.highlight("#no_inline", Yellow)))
			}
		}
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
