package semantics

import (
	"blk/internals"
	"blk/parser"
	"fmt"
	"slices"
)

// This file implemented the type checker & also the type inference
type TypeChecker struct {
	CurrNode      *SymbolInfo
	symbols       *symbolResolver
	collector     *internals.ErrorCollector
	aliasResolver *typeAliasResolver
	Inference     *TypeInference
}

func NewTypeChecker(errCollector *internals.ErrorCollector) *TypeChecker {
	symbols := NewSymbolResolver()
	aliasResolver := NewTypeAliasResolver(symbols)
	tp := &TypeChecker{
		symbols:       symbols,
		collector:     errCollector,
		aliasResolver: aliasResolver,
		Inference:     NewTypeInference(errCollector, aliasResolver, symbols),
	}

	return tp
}

func (s *TypeChecker) SymbolBuilder(ast *parser.Program) {
	// insert prebuilt func later on
	for _, node := range ast.Statements {
		s.symbolReader(node)
	}

	// check if the entry point is a main function
	// access to the main
	mainFn := s.symbols.current.Store["main"]
	if mainFn.Name != "main" {
		errMsg := ("ERROR: no entry point found, consider creating an entry point called main")
		fmt.Println(errMsg)
		return
	}
}

func (s *TypeChecker) symbolReader(node parser.Statement) {
	switch node := node.(type) {
	case *parser.LetStatement:
		s.visitVarDCL(node)
	case *parser.FunctionStatement:
		s.visitFuncDCL(node)
	case *parser.StructStatement:
		s.visitStructDCL(node)
	case *parser.TypeStatement:
		s.visitTypeDCL(node)
	case *parser.WhileStatement:
		s.visitWhileLoopDCL(node)
	case *parser.ForStatement:
		s.visitForLoopDCL(node)
	case *parser.ReturnStatement:
		s.visitReturnDCL(node)
	case *parser.ScopeStatement:
		s.visitScopeDCL(node)
	default:
		// type is an expression statement
		stmt := node.(*parser.ExpressionStatement)
		s.symbolReaderExpression(stmt.Expression)
	}
}

func (s *TypeChecker) symbolReaderExpression(node parser.Expression) {
	switch expr := node.(type) {
	case *parser.Identifier:
		s.visitIdentifier(expr)
	case *parser.CallExpression:
		s.visitCallExpression(expr)
	case *parser.UnaryExpression:
		s.visitUnaryExpression(expr)
	case *parser.IfExpression:
		s.visitIfExpression(expr)
	case *parser.StructInstanceExpression:
		s.visitStructInstanceExpression(expr)
	case *parser.ArrayLiteral:
		s.visitArrayLiteral(expr)
	case *parser.MapLiteral:
		s.visitMapLiteral(expr)
	case *parser.IndexExpression:
		s.visitIndexExpression(expr)
	case *parser.MemberShipExpression:
		s.visitMemberShipAccess(expr)

	default:
	}
}

func (s *TypeChecker) visitFuncDCL(node *parser.FunctionStatement) {
	sym := &SymbolInfo{
		Name:     node.Name.Value,
		Kind:     SymbolFunc,
		Depth:    s.symbols.current.Depth,
		DeclNode: node,
		Type:     node.ReturnType,
	}

	// check if the name doesn't collide with pre-built function
	if _, isMatching := builtInFunctions[sym.Name]; isMatching {
		errMsg := fmt.Sprintf("ERROR: fn %v is a pre-built function, consider renaming your function to something else", sym.Name)
		s.collector.Add(s.collector.Error(node.Name.Token, errMsg))
	}

	_, ok := s.symbols.Resolve(sym.Name)

	if ok {
		errMsg := fmt.Sprintf("ERROR:fn %v is already declared, consider removing the duplicate", sym.Name)
		s.collector.Add(s.collector.Error(node.Token, errMsg))
	}

	s.CurrNode = sym
	s.Inference.CurrSymbol = sym
	parentScope := s.symbols.EnterScope()
	// checks if the arg type is declared or not
	for _, arg := range node.Args {
		argType := arg.Type
		s.visitFieldType(argType)
		// define the args as identifiers in the same symbol tab of the block
		sym := &SymbolInfo{
			Name:     arg.Value,
			DeclNode: arg,
			Kind:     SymbolIdentifier,
			Depth:    s.symbols.current.Depth,
			Type:     arg.Type.(parser.Type),
		}
		s.symbols.Define(sym.Name, sym)
	}

	if node.Body != nil {
		block := node.Body
		// save the return type for the func
		currentFunction := sym

		if len(block.Body) == 0 {
			if currentFunction.Kind == SymbolFunc && currentFunction.Type.String() != "void" {
				errMsg := fmt.Sprintf("ERROR: a function that has (%v) as return type, needs always a return statement", currentFunction.Type)
				tok := currentFunction.Type.GetToken()
				tok.Text = currentFunction.Type.String()
				tok.Col -= len(tok.Text) + 1
				s.collector.Add(s.collector.Error(tok, errMsg))
			}
		} else {
			for idx, nd := range block.Body {
				s.symbolReader(nd)
				// check happens only on the last instruction
				if currentFunction.Kind == SymbolFunc && idx == len(block.Body)-1 && currentFunction.Type.String() != "void" {
					if _, ok := nd.(*parser.ReturnStatement); !ok {
						errMsg := fmt.Sprintf("ERROR: a function that has (%v) as return type, needs always a return statement", currentFunction.Type)
						s.collector.Add(s.collector.Error(nd.GetToken(), errMsg))
					}
				}
			}
		}
	}

	s.symbols.ExitScope(parentScope)
	s.visitFieldType(node.ReturnType)

	// entry point function
	if sym.Name == "main" {
		// the return type needs to be void explicitly
		if node.ReturnType.String() != "void" {
			errMsg := fmt.Sprintf("ERROR: fn (%v) return type needs to be void", sym.Name)
			s.collector.Add(s.collector.Error(node.ReturnType.GetToken(), errMsg))
		}
	}

	s.symbols.Define(sym.Name, sym)
}

func (s *TypeChecker) visitVarDCL(node *parser.LetStatement) {
	kind := SymbolLet

	isMutable := false

	if node.Token.Text == "var" {
		kind = SymbolVar
		isMutable = true
	}

	sym := &SymbolInfo{
		Name:      node.Name.Value,
		Kind:      kind,
		Depth:     s.symbols.current.Depth,
		IsMutable: isMutable,
		DeclNode:  node,
		Type:      node.ExplicitType,
	}

	_, ok := s.symbols.current.Store[node.Name.Value]
	if ok {
		errMsg := fmt.Sprintf("ERROR: %v identifier is already declared", node.Name.Value)
		s.collector.Add(s.collector.Error(node.Token, errMsg))
	}
	// this is for to save the last current node if we're in function scope
	tempCurr := s.CurrNode
	s.CurrNode = sym
	s.Inference.CurrSymbol = tempCurr
	s.visitFieldType(node.ExplicitType)
	s.symbolReaderExpression(node.Value)

	expectedType := internals.ParseToNodeType(s.aliasResolver.normalizeType(node.ExplicitType))
	gotType := s.Inference.inferAssociatedValueType(node.Value)

	switch ept := expectedType.(type) {
	case *parser.NodeType:
		switch ift := gotType.(type) {
		case *parser.NodeType:
			if isMatching, errNode := internals.DeepEqualOnNodeType(ept, ift); !isMatching {
				errMsg := ""
				if gotType.String() == "nil" {
					errMsg = fmt.Sprintf("ERROR: type mismatch, expected %v, got %v, check nest level of the array", ept, ift)
				} else {
					errMsg = fmt.Sprintf("ERROR: type mismatch on (%v), original type (%v)", errNode, ept)
				}
				tok := node.Value.GetToken()
				tok.Text = node.Value.String()
				s.collector.Add(s.collector.Error(tok, errMsg))
			}
		default:
			// fall through
			errMsg := ""
			if gotType == nil {
				errMsg = fmt.Sprintf("ERROR: type mismatch, expected %v, got %v", ept, gotType)
			} else {
				if gotType.String() == "nil" {
					errMsg = fmt.Sprintf("ERROR: type mismatch, expected %v, got %v, check nest level of the array", ept, ift)
				} else {
					errMsg = fmt.Sprintf("ERROR: type mismatch on (%v), original type (%v)", expectedType, ept)
				}
			}
			tok := node.Value.GetToken()
			tok.Text = node.Value.String()
			s.collector.Add(s.collector.Error(tok, errMsg))
		}
	case *parser.MapType:
		switch ift := gotType.(type) {
		case *parser.MapType:
			if isMatching, errNode := internals.DeepEqualOnMapType(ept, ift); !isMatching {
				errMsg := ""
				if gotType.String() == "nil" {
					errMsg = fmt.Sprintf("ERROR: type mismatch, expected %v, got %v, check nest level of the array", ept, ift)
				} else {
					errMsg = fmt.Sprintf("ERROR: type mismatch on %v, expected type %v", errNode, ept)
				}
				tok := node.Value.GetToken()
				tok.Text = node.Value.String()
				s.collector.Add(s.collector.Error(tok, errMsg))
			}
		default:
			errMsg := ""
			if gotType == nil {
				errMsg = fmt.Sprintf("ERROR: type mismatch, expected %v, got %v", ept, gotType)
			} else {
				if gotType.String() == "nil" {
					errMsg = fmt.Sprintf("ERROR: type mismatch, expected %v, got %v, check nest level of the array", ept, ift)
				} else {
					errMsg = fmt.Sprintf("ERROR: type mismatch on %v, expected type %v", gotType, ept)
				}
			}
			tok := node.Value.GetToken()
			tok.Text = node.Value.String()
			s.collector.Add(s.collector.Error(tok, errMsg))
		}
	}

	s.CurrNode = tempCurr
	s.Inference.CurrSymbol = tempCurr
	sym.Type = node.ExplicitType
	s.symbols.Define(sym.Name, sym)
}

func (s *TypeChecker) visitStructDCL(node *parser.StructStatement) *SymbolInfo {

	sym := &SymbolInfo{
		Name:     node.Name.Value,
		Kind:     SymbolStruct,
		Depth:    s.symbols.current.Depth,
		DeclNode: node,
		Type: &parser.NodeType{
			Type: node.Name.Value,
		},
	}

	_, ok := s.symbols.Resolve(sym.Name)

	if ok {
		errMsg := fmt.Sprintf("ERROR: %v identifier is already declared", sym.Name)
		s.collector.Add(s.collector.Error(node.Token, errMsg))
	}

	s.symbols.Define(sym.Name, sym)
	if len(node.Body) > 0 {
		scope := s.symbols.EnterScope()
		for _, field := range node.Body {
			// check for field redundancy
			fieldName := field.Key.Value
			_, ok := s.symbols.Resolve(fieldName)
			if ok {
				errMsg := fmt.Sprintf("ERROR: ( %v ) key is already declared, attempt to re-declare", fieldName)
				s.collector.Add(s.collector.Error(field.Key.Token, errMsg))
				return nil
			} else {
				s.symbols.Define(fieldName, &SymbolInfo{
					Name:     field.Key.Value,
					Kind:     SymbolIdentifier,
					Depth:    s.symbols.current.Depth,
					DeclNode: field.Key,
				})
			}

			// check for if the type is custom type
			// and if it is check if it already in the table or not
			fieldType := field.Value
			s.visitFieldType(fieldType)
		}
		s.symbols.ExitScope(scope)
	}
	return sym
}

func (s *TypeChecker) visitFieldType(fieldType parser.Expression) {

	switch tp := fieldType.(type) {
	case *parser.NodeType:
		if _, ok := parser.AtomicTypes[tp.Type]; !ok {
			if tp.Type != "array" {
				_, exist := s.symbols.Resolve(tp.Type)
				if !exist {
					errMsg := fmt.Sprintf("ERROR: type ( %v ) needs to be declared before it gets used", tp.Type)
					s.collector.Add(s.collector.Error(tp.Token, errMsg))
				}
			}
		}

		// check for this in a recursive manner if the tp.ChildNode != nil
		if tp.ChildType != nil {
			s.visitFieldType(tp.ChildType)
		}

	case *parser.MapType:
		if _, ok := parser.AtomicTypes[tp.Type]; !ok {
			if tp.Type != "map" {
				_, exist := s.symbols.Resolve(tp.Type)
				if !exist {
					errMsg := fmt.Sprintf("ERROR: type ( %v ) needs to be declared before it gets used", tp.Type)
					s.collector.Add(s.collector.Error(tp.Token, errMsg))
				}

			}
		}

		if tp.Left != nil {
			s.visitFieldType(tp.Left)
		}

		if tp.Right != nil {
			s.visitFieldType(tp.Right)
		}

	default:
		// nothing
	}

}

func (s *TypeChecker) visitTypeDCL(node *parser.TypeStatement) {
	sym := &SymbolInfo{
		Name:     node.Name.Value,
		Kind:     SymbolType,
		Depth:    s.symbols.current.Depth,
		DeclNode: node,
		Type:     node.Value,
	}

	_, ok := s.symbols.Resolve(sym.Name)

	if ok {
		tok := node.Name.Token
		errMsg := fmt.Sprintf("ERROR: %v identifier is already declared", node.Name.Value)
		s.collector.Add(s.collector.Error(tok, errMsg))
	}

	s.visitFieldType(node.Value)
	s.symbols.Define(sym.Name, sym)

}

func (s *TypeChecker) visitBlockDCL(block *parser.BlockStatement) {
	// mean the body of the current scope is empty
	parentScope := s.symbols.EnterScope()

	// save the return type for the func
	currentFunction := s.CurrNode

	if len(block.Body) == 0 {
		if currentFunction.Kind == SymbolFunc && currentFunction.Type.String() != "void" {
			errMsg := fmt.Sprintf("ERROR: a function that has (%v) as return type, needs always a return statement", currentFunction.Type)
			tok := currentFunction.Type.GetToken()
			tok.Text = currentFunction.Type.String()
			tok.Col -= len(tok.Text) + 1
			s.collector.Add(s.collector.Error(tok, errMsg))
		}
	} else {
		for idx, nd := range block.Body {
			s.symbolReader(nd)
			// check happens only on the last instruction
			if currentFunction.Kind == SymbolFunc && idx == len(block.Body)-1 && currentFunction.Type.String() != "void" {
				if _, ok := nd.(*parser.ReturnStatement); !ok {
					errMsg := fmt.Sprintf("ERROR: a function that has (%v) as return type, needs always a return statement", currentFunction.Type)
					s.collector.Add(s.collector.Error(nd.GetToken(), errMsg))
				}
			}
		}
	}

	s.symbols.ExitScope(parentScope)
}

func (s *TypeChecker) visitWhileLoopDCL(node *parser.WhileStatement) {
	// check if the condition
	condition := node.Condition

	switch cnd := condition.(type) {
	case *parser.Identifier:
		// later on check if the identifier will get evaluated to a boolean
		s.visitIdentifier(cnd)
	case *parser.UnaryExpression:
		s.visitUnaryExpression(cnd)
	default:
		// do nothing
	}

	// check the body
	s.visitBlockDCL(node.Body)
}

func (s *TypeChecker) visitForLoopDCL(node *parser.ForStatement) {
	for _, ident := range node.Identifiers {
		sym := &SymbolInfo{
			Name:     ident.Value,
			DeclNode: ident,
			Kind:     SymbolIdentifier,
			Depth:    s.symbols.current.Depth,
			// add type of for loop identifiers
		}

		s.symbols.Define(sym.Name, sym)
	}

	// check for the target if it already existing
	s.symbolReaderExpression(node.Target)
	// check for the body
	s.visitBlockDCL(node.Body)
}

func (s *TypeChecker) visitReturnDCL(node *parser.ReturnStatement) {
	if s.symbols.current.Depth == 0 {
		// means it is on the global scope not in a function
		errMsg := "ERROR: return statement, can't be on the global scope, needs to be inside of a function"
		s.collector.Add(s.collector.Error(node.Token, errMsg))
	}

	// check if the associated return value on the return statement, is the same as the return value of the function
	functionReturnType := s.Inference.inferAssociatedValueType(node.ReturnValue)
	returnType := internals.ParseToNodeType(s.aliasResolver.normalizeType(s.CurrNode.Type))

	switch ept := returnType.(type) {
	case *parser.NodeType:
		switch ift := functionReturnType.(type) {
		case *parser.NodeType:
			if isMatching, errNode := internals.DeepEqualOnNodeType(ept, ift); !isMatching {
				errMsg := ""
				if functionReturnType.String() == "nil" {
					errMsg = fmt.Sprintf("ERROR: type mismatch, expected %v, got %v, check nest level of the array", ept, ift)
				} else {
					errMsg = fmt.Sprintf("ERROR: type mismatch on %v, expected type %v", errNode, ept)
				}
				tok := node.ReturnValue.GetToken()
				tok.Text = node.ReturnValue.String()
				s.collector.Add(s.collector.Error(tok, errMsg))
			}
		default:
			errMsg := ""
			if functionReturnType.String() == "nil" {
				errMsg = fmt.Sprintf("ERROR: type mismatch, expected %v, got %v, check nest level of the array", ept, ift)
			} else {
				errMsg = fmt.Sprintf("ERROR: type mismatch on %v, expected type %v", functionReturnType, ept)
			}
			tok := node.ReturnValue.GetToken()
			tok.Text = node.ReturnValue.String()
			s.collector.Add(s.collector.Error(tok, errMsg))
		}
	case *parser.MapType:
		switch ift := functionReturnType.(type) {
		case *parser.MapType:
			if isMatching, errNode := internals.DeepEqualOnMapType(ept, ift); !isMatching {
				errMsg := ""
				if functionReturnType.String() == "nil" {
					errMsg = fmt.Sprintf("ERROR: type mismatch, expected %v, got %v, check nest level of the array", ept, ift)
				} else {
					errMsg = fmt.Sprintf("ERROR: type mismatch on %v, expected type %v", errNode, ept)
				}
				tok := node.ReturnValue.GetToken()
				tok.Text = node.ReturnValue.String()
				s.collector.Add(s.collector.Error(tok, errMsg))
			}
		default:
			errMsg := ""
			if functionReturnType.String() == "nil" {
				errMsg = fmt.Sprintf("ERROR: type mismatch, expected %v, got %v, check nest level of the array", ept, ift)
			} else {
				errMsg = fmt.Sprintf("ERROR: type mismatch on %v, expected type %v", functionReturnType, ept)
			}
			tok := node.ReturnValue.GetToken()
			tok.Text = node.ReturnValue.String()
			s.collector.Add(s.collector.Error(tok, errMsg))
		}
	default:
		// fall through otherwise
		panic("ERROR: UNREACHABLE, VISIT RETURN DCL")
	}
}

func (s *TypeChecker) visitScopeDCL(node *parser.ScopeStatement) {
	if node.Body != nil {
		s.visitBlockDCL(node.Body)
	}
}

func (s *TypeChecker) visitCallExpression(expr *parser.CallExpression) {
	functionName := expr.Function.Value

	function, isMatched := s.symbols.Resolve(functionName)

	_, isBuiltInFunc := builtInFunctions[functionName]

	if !isMatched && !isBuiltInFunc {
		errMsg := fmt.Sprintf("ERROR: (%v) function, needs to be declared before it get called", expr.Function.Value)
		s.collector.Add(s.collector.Error(expr.Token, errMsg))
		return
	}

	dclNode := function.DeclNode.(*parser.FunctionStatement)
	returnValue := dclNode.ReturnType
	s.visitFieldType(returnValue)

	// check if same number of the args provided is the same
	args := expr.Args
	if len(args) < len(dclNode.Args) {
		errMsg := "ERROR: need to pass all the args into the function call"
		tok := dclNode.Args[len(args)].Token
		expr.Token.Col = expr.Token.Col + len(expr.Token.Text)
		expr.Token.Text = tok.Text
		s.collector.Add(s.collector.Error(expr.Token, errMsg))
		return
	}

	if len(args) > len(dclNode.Args) {
		errMsg := "ERROR: function call is receiving more args than it should, consider removing them, or add them into the function signature"
		tok := parser.Token{}
		startIdx := len(dclNode.Args)

		for idx, arg := range args[startIdx:] {

			if expr, ok := arg.(parser.Node); ok {
				tok.Text += expr.String()
				tok.Row = expr.GetToken().Row
				if idx == 0 {
					tok.Col = expr.GetToken().Col
				}
			}

			if idx+1 <= len(args)-startIdx-1 {
				tok.Text += ", "
			}
		}
		s.collector.Add(s.collector.Error(tok, errMsg))
		return
	}

	// check if the args of the call expr, already exist
	// TODO: check also if there associated type is the same as the type of args on the function signature
	for idx, arg := range args {
		switch ag := arg.(type) {
		case *parser.Identifier:
			s.visitIdentifier(ag)
			// fall through
		case *parser.MemberShipExpression:
			s.visitMemberShipAccess(ag)

		default:
			// handle only atomic types (strings, booleans, floats, ints)
			returnType := s.Inference.inferAssociatedValueType(ag)
			tp := dclNode.Args[idx].Type
			s.argTypeChecker(tp.String(), returnType.String(), arg)
		}

		returnType := s.Inference.inferAssociatedValueType(arg)
		tp := dclNode.Args[idx].Type
		s.argTypeChecker(tp.String(), returnType.String(), arg)
	}

}

func (s *TypeChecker) argTypeChecker(tp, returnType string, arg parser.Expression) {
	if tp != returnType {
		errMsg := fmt.Sprintf("ERROR: type mismatch, expected %v, got %v", tp, returnType)
		tok := arg.GetToken()
		tok.Text = arg.String()
		s.collector.Add(s.collector.Error(tok, errMsg))
	}
}

func (s *TypeChecker) visitUnaryExpression(expr *parser.UnaryExpression) {
	// check if the type is boolean or not
	if s.CurrNode.Type.(*parser.NodeType).Type == parser.BoolType && expr.Operator == "-" {
		errMsg := "ERROR: can't use operator (-) with boolean types, only operator (!) is allowed"
		s.collector.Add(s.collector.Error(expr.Token, errMsg))
		return
	}

	s.symbolReaderExpression(expr.Right)
}

func (s *TypeChecker) visitIfExpression(expr *parser.IfExpression) {
	conditionExpression := expr.Condition

	switch cExpr := conditionExpression.(type) {
	case *parser.UnaryExpression:
		s.visitUnaryExpression(cExpr)
	case *parser.Identifier:
		s.visitIdentifier(cExpr)
	case *parser.IndexExpression:
		s.visitIndexExpression(cExpr)
	default:
		//
	}

	// check the body of if
	s.visitBlockDCL(expr.Consequence)

	// check the body of the else, if it exists
	switch alt := expr.Alternative.(type) {
	case *parser.BlockStatement:
		s.visitBlockDCL(alt)
	case *parser.IfExpression:
		s.visitIfExpression(alt)
	default:
		// Do nothing
	}
}

func (s *TypeChecker) visitIdentifier(expr *parser.Identifier) *SymbolInfo {
	// if (identifier) check if it declared or not
	ident, isMatched := s.symbols.Resolve(expr.Value)

	if !isMatched {
		errMsg := ("ERROR: identifier, needs to be declared before it gets used")
		s.collector.Add(s.collector.Error(expr.Token, errMsg))
		return nil
	}

	return ident
}

func (s *TypeChecker) visitStructInstanceExpression(expr *parser.StructInstanceExpression) {

	identifier, ok := expr.Left.(*parser.Identifier)

	if ok {
		structDef, isMatched := s.symbols.Resolve(identifier.Value)

		if !isMatched {
			errMsg := fmt.Sprintf("ERROR: struct (%v) needs to be defined, before creating instances of it", identifier.Value)
			s.collector.Add(s.collector.Error(identifier.Token, errMsg))
			return
		}

		body := expr.Body

		keys := &parser.StructStatement{}

		switch strcDf := structDef.DeclNode.(type) {
		case *parser.StructStatement:
			keys = strcDf
		case *parser.TypeStatement:
			keys = s.visitIdentifier(
				&parser.Identifier{
					Value: strcDf.Value.(*parser.NodeType).Type,
				},
			).DeclNode.(*parser.StructStatement)
		}

		// means that some fields are left out of the having an associated value
		for _, field := range keys.Body {
			// find the key in the struct instance
			idx := slices.IndexFunc(body, func(f parser.FieldInstance) bool {
				return f.Key.Value == field.Key.Value
			})

			if idx == -1 {
				errMsg := fmt.Sprintf("ERROR: field (%v) needs to be instantiated with a value, cause it exists on the struct definition", field.Key.Value)
				s.collector.Add(s.collector.Error(field.Key.Token, errMsg))
			}
		}

		for _, field := range body {
			// find the key in the struct instance
			idx := slices.IndexFunc(keys.Body, func(f parser.Field) bool {
				return f.Key.Value == field.Key.Value
			})

			if idx == -1 {
				errMsg := fmt.Sprintf("ERROR: field (%v) doesn't exist on the struct definition, either add it to the definition or remove the field from the instance", field.Key.Value)
				s.collector.Add(s.collector.Error(field.Key.Token, errMsg))
			}
		}
	}
}

func (s *TypeChecker) visitArrayLiteral(expr *parser.ArrayLiteral) {
	elements := expr.Elements

	for _, elem := range elements {
		s.symbolReaderExpression(elem)
	}
}

func (s *TypeChecker) visitMapLiteral(expr *parser.MapLiteral) {

	pairs := expr.Pairs

	for key, value := range pairs {
		switch k := key.(type) {
		case *parser.Identifier:
			s.visitIdentifier(k)
		case *parser.CallExpression:
			s.visitCallExpression(k)
		case *parser.IndexExpression:
			s.visitIndexExpression(k)

		default:
			// panic(fmt.Sprintf("ERROR: %v ain't supported in map literal (key check)", k))
		}
		s.symbolReaderExpression(value)
	}
}

func (s *TypeChecker) visitIndexExpression(expr *parser.IndexExpression) {
	// check if the lest side is a valid
	tok := parser.Token{}
	errMsg := ""
	switch lf := expr.Left.(type) {
	case *parser.Identifier:
		s.visitIdentifier(lf)
	case *parser.ArrayLiteral:
		s.visitArrayLiteral(lf)
	case *parser.CallExpression:
		s.visitCallExpression(lf)
	case *parser.IfExpression:
		errMsg = "ERROR: can't use an if expression as left side of index expression"
		tok = lf.Token
		tok.Text = lf.String()
	case *parser.BinaryExpression:
		errMsg = "ERROR: can't use a binary expression as left side of index expression, cause it evaluates to a boolean"
		// construct the token
		tok = lf.Token
		lf.Token.Col -= 1
		tok.Text = lf.String()
	case *parser.UnaryExpression:
		errMsg = "ERROR: can't use a unary expression as left side of index expression, cause it evaluates to a boolean"
		tok = lf.Token
		tok.Text = lf.String()
	case *parser.StructInstanceExpression:
		errMsg = "ERROR: can't use a struct instance as left side of index expression"
		tok = lf.Token
		tok.Text = lf.String()
	default:
		// nothing
	}

	if len(tok.Text) > 0 {
		s.collector.Add(s.collector.Error(tok, errMsg))
	}

	switch rf := expr.Index.(type) {
	case *parser.Identifier:
		s.visitIdentifier(rf)
	case *parser.CallExpression:
		s.visitCallExpression(rf)
	case *parser.IndexExpression:
		s.visitIndexExpression(rf)
	case *parser.BinaryExpression:
		errMsg = "ERROR: can't use a binary expression, cause it evaluates to a boolean"
		// construct the token
		rf.Token.Text = rf.String()
		tok = rf.Token
		tok.Text = rf.String()
	case *parser.UnaryExpression:
		errMsg = "ERROR: can't use a unary expression, cause it evaluates to a boolean"
		tok = rf.Token
		tok.Text = rf.String()
	case *parser.IfExpression:
		errMsg = "ERROR: can't use an if expression as index, index can only be an int"
		tok = rf.Token
		tok.Text = rf.String()
	default:

	}

	if len(tok.Text) > 0 {
		s.collector.Add(s.collector.Error(tok, errMsg))
	}
}

func (s *TypeChecker) visitMemberShipAccess(expr *parser.MemberShipExpression) {
	s.symbolReaderExpression(expr.Object)
	s.symbolReaderExpression(expr.Property)
}
