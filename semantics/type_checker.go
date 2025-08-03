package semantics

import (
	"blk/ast"
	"blk/internals"
	"blk/lexer"
	"blk/parser"
	"fmt"
	"slices"
)

// This file implemented the type checker & also the type inference
type TypeChecker struct {
	currNode      *SymbolInfo
	symbols       *symbolResolver
	collector     *internals.ErrorCollector
	aliasResolver *typeAliasResolver
	inference     *TypeInference
}

func NewTypeChecker(errCollector *internals.ErrorCollector) *TypeChecker {
	symbols := NewSymbolResolver()
	aliasResolver := NewTypeAliasResolver(symbols)
	tp := &TypeChecker{
		symbols:       symbols,
		collector:     errCollector,
		aliasResolver: aliasResolver,
		inference:     NewTypeInference(errCollector, aliasResolver, symbols),
	}

	return tp
}

func (s *TypeChecker) SymbolBuilder(ast *ast.Program) {
	// insert prebuilt func later on
	for _, node := range ast.Statements {
		s.symbolReader(node)
	}

	// check if the entry point is a main function
	mainFn := s.symbols.current.Store["main"]
	if mainFn.Name != "main" {
		errMsg := ("ERROR: no entry point found, consider creating an entry point called main")
		fmt.Println(errMsg)
		return
	}
}

func (s *TypeChecker) symbolReader(node ast.Statement) {
	switch node := node.(type) {
	case *ast.VarDeclaration:
		s.visitVarDCL(node)
	case *ast.WhileStatement:
		s.visitWhileLoopDCL(node)
	case *ast.ForStatement:
		s.visitForLoopDCL(node)
	case *ast.ReturnStatement:
		s.visitReturnDCL(node)
	case *ast.ScopeStatement:
		s.visitScopeDCL(node)
	case *ast.ImportStatement:
		// TODO: implement later

	default:
		// type is an expression statement
		stmt := node.(*ast.ExpressionStatement)
		s.symbolReaderExpression(stmt.Expression)
	}
}

func (s *TypeChecker) symbolReaderExpression(node ast.Expression) {
	switch expr := node.(type) {
	case *ast.Identifier:
		s.visitIdentifier(expr)
	case *ast.CallExpression:
		s.visitCallExpression(expr)
	case *ast.UnaryExpression:
		s.visitUnaryExpression(expr)
	case *ast.BinaryExpression:
		s.visitBinaryExpression(expr)
	case *ast.IfExpression:
		s.visitIfExpression(expr)
	case *ast.StructInstanceExpression:
		s.visitStructInstanceExpression(expr)
	case *ast.ArrayLiteral:
		s.visitArrayLiteral(expr)
	case *ast.MapLiteral:
		s.visitMapLiteral(expr)
	case *ast.IndexExpression:
		s.visitIndexExpression(expr)
	case *ast.MemberShipExpression:
		s.visitMemberShipAccess(expr)
	// TODO: add missing expressions
	case *ast.FunctionExpression:
		s.visitFuncDCL(expr)
	case *ast.StructExpression:
		s.visitStructDCL(expr)

	default:
	}
}

func (s *TypeChecker) visitFuncDCL(expr *ast.FunctionExpression) {

	// check the args and body
	newScope := s.symbols.EnterScope()

	// checks if the arg type is declared or not
	for _, arg := range expr.Args {
		argType := arg.Type
		s.visitFieldType(argType)
		// define the args as identifiers in the same symbol tab of the block
		sym := &SymbolInfo{
			Name:     arg.Value,
			DeclNode: arg,
			Kind:     SymbolIdentifier,
			Depth:    s.symbols.current.Depth,
			Type:     arg.Type.(ast.Type),
		}
		s.symbols.Define(sym.Name, sym)
	}

	if expr.Body != nil {
		block := expr.Body
		// save the return type for the func

		if len(block.Body) == 0 {
			if expr.ReturnType.String() != "void" {
				errMsg := fmt.Sprintf("ERROR: a function that has (%v) as return type, needs always a return statement", expr.ReturnType)
				tok := expr.ReturnType.GetToken()
				tok.Text = expr.ReturnType.String()
				tok.Col -= len(tok.Text) + 1
				s.collector.Add(s.collector.Error(tok, errMsg))
			}
		} else {
			for idx, nd := range block.Body {
				s.symbolReader(nd)
				// check happens only on the last instruction
				if idx == len(block.Body)-1 && expr.ReturnType.String() != "void" {
					if _, ok := nd.(*ast.ReturnStatement); !ok {
						errMsg := fmt.Sprintf("ERROR: a function that has (%v) as return type, needs always a return statement", expr.ReturnType)
						s.collector.Add(s.collector.Error(nd.GetToken(), errMsg))
					}
				}
			}
		}
	}

	s.symbols.ExitScope(newScope)
	s.visitFieldType(expr.ReturnType)

}

func (s *TypeChecker) visitVarDCL(node *ast.VarDeclaration) {
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

	if _, ok := node.Value.(*ast.StructInstanceExpression); ok {
		sym.Type = node.Value
		s.symbols.Define(sym.Name, sym)
	}

	// this is for to save the last current node if we're in function scope
	tempCurr := s.currNode
	s.currNode = sym
	s.inference.currSymbol = sym
	s.visitFieldType(node.ExplicitType)
	s.symbolReaderExpression(node.Value)
	gotType := s.inference.inferAssociatedValueType(node.Value)

	// if the explicit type is not nil we do the comparison between the inferred result and the type set by the programmer
	if node.ExplicitType != nil {
		expectedType := internals.ParseToNodeType(s.aliasResolver.normalizeType(node.ExplicitType))
		s.typeEquals(expectedType, gotType, node.Value)
	}

	s.currNode = tempCurr
	s.inference.currSymbol = tempCurr
	s.symbols.Define(sym.Name, sym)
}

func (s *TypeChecker) visitStructDCL(expr *ast.StructExpression) {
	// we define the struct here, in case if it is used in an inner field(s)
	if len(expr.Body) > 0 {
		newScope := s.symbols.EnterScope()
		for _, field := range expr.Body {
			// check for field redundancy
			fieldName := field.Key.Value
			_, ok := s.symbols.Resolve(fieldName)
			if ok {
				errMsg := fmt.Sprintf("ERROR: ( %v ) key is already declared, attempt to re-declare", fieldName)
				s.collector.Add(s.collector.Error(field.Key.Token, errMsg))
			} else {
				s.symbols.Define(fieldName, &SymbolInfo{
					Name:     field.Key.Value,
					Kind:     SymbolIdentifier,
					Depth:    s.symbols.current.Depth,
					DeclNode: field.Key,
				})
			}
			switch ftp := field.Value.(type) {
			case *ast.Identifier:
				// do something
				fmt.Println(ftp)
			case *ast.FunctionExpression:
				// do something
				s.currNode = &SymbolInfo{
					Name:     fieldName,
					Kind:     SymbolFunc,
					Depth:    s.symbols.current.Depth,
					DeclNode: field.Value,
					Type:     ftp.ReturnType,
				}
				s.inference.currSymbol = s.currNode
				s.visitFuncDCL(ftp)

			case *ast.MapType, *ast.NodeType:
				s.visitFieldType(ftp)
			default:
				errMsg := fmt.Sprintf("ERROR: can't use %v as type of field in a given struct", ftp)
				s.collector.Add(s.collector.Error(field.Key.Token, errMsg))
			}
			// check for if the type is custom type
		}
		s.symbols.ExitScope(newScope)
	}
}

func (s *TypeChecker) visitFieldType(fieldType ast.Expression) {

	switch tp := fieldType.(type) {
	case *ast.NodeType:
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

	case *ast.MapType:
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

// func (s *TypeChecker) visitTypeDCL(node *ast.TypeStatement) {
// 	sym := &SymbolInfo{
// 		Name:     node.Name.Value,
// 		Kind:     SymbolType,
// 		Depth:    s.symbols.current.Depth,
// 		DeclNode: node,
// 		Type:     node.Value,
// 	}

// 	_, ok := s.symbols.Resolve(sym.Name)

// 	if ok {
// 		tok := node.Name.Token
// 		errMsg := fmt.Sprintf("ERROR: %v identifier is already declared", node.Name.Value)
// 		s.collector.Add(s.collector.Error(tok, errMsg))
// 	}

// 	s.visitFieldType(node.Value)
// 	s.symbols.Define(sym.Name, sym)

// }

func (s *TypeChecker) visitBlockDCL(block *ast.BlockStatement) {

	// save the return type for the func
	currentFunction := s.currNode

	// means the body of the current scope is empty
	if len(block.Body) == 0 {
		if currentFunction.Kind == SymbolFunc && currentFunction.Type.String() != "void" {
			errMsg := fmt.Sprintf("ERROR: a function that has (%v) as return type, needs always a return statement", currentFunction.Type)
			tok := currentFunction.Type.GetToken()
			tok.Text = currentFunction.Type.String()
			tok.Col -= len(tok.Text) + 1
			s.collector.Add(s.collector.Error(tok, errMsg))
		}
	} else {
		newScope := s.symbols.EnterScope()
		for idx, nd := range block.Body {
			s.symbolReader(nd)
			// check happens only on the last instruction
			if currentFunction.Kind == SymbolFunc && idx == len(block.Body)-1 && currentFunction.Type.String() != "void" {
				if _, ok := nd.(*ast.ReturnStatement); !ok {
					errMsg := fmt.Sprintf("ERROR: a function that has (%v) as return type, needs always a return statement", currentFunction.Type)
					s.collector.Add(s.collector.Error(nd.GetToken(), errMsg))
				}
			}
		}
		s.symbols.ExitScope(newScope)
	}

}

func (s *TypeChecker) visitWhileLoopDCL(node *ast.WhileStatement) {
	// check if the condition
	condition := node.Condition

	switch cnd := condition.(type) {
	case *ast.Identifier:
		// later on check if the identifier will get evaluated to a boolean
		s.visitIdentifier(cnd)
	case *ast.UnaryExpression:
		s.visitUnaryExpression(cnd)
	default:
		// do nothing
	}

	// check the body
	s.visitBlockDCL(node.Body)
}

func (s *TypeChecker) visitForLoopDCL(node *ast.ForStatement) {
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

func (s *TypeChecker) visitReturnDCL(node *ast.ReturnStatement) {
	if s.symbols.current.Depth == 0 {
		// means it is on the global scope not in a function
		errMsg := "ERROR: return statement, can't be on the global scope, needs to be inside of a function"
		s.collector.Add(s.collector.Error(node.Token, errMsg))
	}

	// check if the associated return value on the return statement, is the same as the return value of the function
	returnType := internals.ParseToNodeType(s.aliasResolver.normalizeType(s.currNode.Type))
	s.symbolReaderExpression(node.ReturnValue)
	functionReturnType := s.inference.inferAssociatedValueType(node.ReturnValue)

	s.typeEquals(returnType, functionReturnType, node.ReturnValue)
}

func (s *TypeChecker) visitScopeDCL(node *ast.ScopeStatement) {
	// TODO: check if the name already exists here later on
	if node.Body != nil {
		s.visitBlockDCL(node.Body)
	}
}

func (s *TypeChecker) visitCallExpression(expr *ast.CallExpression) {
	functionName := expr.Function.Value

	function, isMatched := s.symbols.Resolve(functionName)

	_, isBuiltInFunc := builtInFunctions[functionName]

	if !isMatched && !isBuiltInFunc {
		errMsg := fmt.Sprintf("ERROR: (%v) function, needs to be declared before it get called", expr.Function.Value)
		s.collector.Add(s.collector.Error(expr.Token, errMsg))
		return
	}

	dclNode := function.DeclNode.(*ast.VarDeclaration).Value.(*ast.FunctionExpression)
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
		tok := lexer.Token{}
		startIdx := len(dclNode.Args)

		for idx, arg := range args[startIdx:] {

			if expr, ok := arg.(ast.Node); ok {
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

	// checks if the args of the call expr, already exist
	// and also tries to infer the type and do the comparison between the definition and the provided ones
	for idx, arg := range args {
		switch ag := arg.(type) {
		case *ast.IfExpression:
			errMsg := "ERROR: can't use if expression as an arg of a call function"
			tok := ag.GetToken()
			tok.Text = ag.String()
			s.collector.Add(s.collector.Error(tok, errMsg))

		default:
			// handle only atomic types (strings, booleans, floats, ints)
			returnType := s.inference.inferAssociatedValueType(ag)
			tp := dclNode.Args[idx].Type.(ast.Type)
			s.typeEquals(tp, returnType, arg)
		}
	}

}

func (s *TypeChecker) typeEquals(tp, returnType ast.Type, arg ast.Expression) {

	switch ept := tp.(type) {
	case *ast.NodeType:
		switch ift := returnType.(type) {
		case *ast.NodeType:
			if isMatching, errNode := internals.DeepEqualOnNodeType(ept, ift); !isMatching {
				errMsg := ""
				if returnType.String() == "nil" {
					errMsg = fmt.Sprintf("ERROR: type mismatch, expected %v, got %v, check nest level of the array", ept, ift)
				} else {
					errMsg = fmt.Sprintf("ERROR: type mismatch on (%v), original type (%v)", errNode, ept)
				}
				tok := arg.GetToken()
				tok.Text = arg.String()
				s.collector.Add(s.collector.Error(tok, errMsg))
			}
		default:
			// fall through
			errMsg := ""
			if returnType == nil {
				errMsg = fmt.Sprintf("ERROR: type mismatch, expected %v, got %v", ept, returnType)
			} else {
				if returnType.String() == "nil" {
					errMsg = fmt.Sprintf("ERROR: type mismatch, expected %v, got %v, check nest level of the array", ept, ift)
				} else {
					errMsg = fmt.Sprintf("ERROR: type mismatch on (%v), original type (%v)", tp, ept)
				}
			}
			tok := arg.GetToken()
			tok.Text = arg.String()
			s.collector.Add(s.collector.Error(tok, errMsg))
		}
	case *ast.MapType:
		switch ift := returnType.(type) {
		case *ast.MapType:
			if isMatching, errNode := internals.DeepEqualOnMapType(ept, ift); !isMatching {
				errMsg := ""
				if returnType.String() == "nil" {
					errMsg = fmt.Sprintf("ERROR: type mismatch, expected %v, got %v, check nest level of the array", ept, ift)
				} else {
					errMsg = fmt.Sprintf("ERROR: type mismatch on %v, expected type %v", errNode, ept)
				}
				tok := arg.GetToken()
				tok.Text = arg.String()
				s.collector.Add(s.collector.Error(tok, errMsg))
			}
		default:
			errMsg := ""
			if returnType == nil {
				errMsg = fmt.Sprintf("ERROR: type mismatch, expected %v, got %v", ept, returnType)
			} else {
				if returnType.String() == "nil" {
					errMsg = fmt.Sprintf("ERROR: type mismatch, expected %v, got %v, check nest level of the array", ept, ift)
				} else {
					errMsg = fmt.Sprintf("ERROR: type mismatch on %v, expected type %v", returnType, ept)
				}
			}
			tok := arg.GetToken()
			tok.Text = arg.String()
			s.collector.Add(s.collector.Error(tok, errMsg))
		}
	}
}

func (s *TypeChecker) visitUnaryExpression(expr *ast.UnaryExpression) {
	// check if the type is boolean or not
	if s.currNode.Type.(*ast.NodeType).Type == ast.BoolType && expr.Operator == "-" {
		errMsg := "ERROR: can't use operator (-) with boolean types, only operator (!) is allowed"
		s.collector.Add(s.collector.Error(expr.Token, errMsg))
		return
	}

	errMsg := ""
	switch expr.Right.(type) {
	case *ast.IfExpression:
		errMsg = "ERROR: can't have an if expression as the right side of a unary expression"
	case *ast.ArrayLiteral:
		errMsg = "ERROR: can't have an array literal as the right side of a unary expression"
	case *ast.MapLiteral:
		errMsg = "ERROR: can't have a map literal as the right side of a unary expression"
	default:
		s.symbolReaderExpression(expr.Right)
	}

	if len(errMsg) > 0 {
		tok := expr.Right.GetToken()
		tok.Text = expr.Right.String()
		s.collector.Add(s.collector.Error(tok, errMsg))
	}
}

func (s *TypeChecker) visitBinaryExpression(expr *ast.BinaryExpression) {
	// check if the type is boolean or not
	errMsg := ""
	switch expr.Left.(type) {
	case *ast.IfExpression:
		errMsg = "ERROR: can't have an if expression as the left side of a binary expression"
	case *ast.ArrayLiteral:
		errMsg = "ERROR: can't have an array literal as the left side of a binary expression"
	case *ast.MapLiteral:
		errMsg = "ERROR: can't have a map literal as the left side of a binary expression"
	default:
		s.symbolReaderExpression(expr.Right)
	}

	if len(errMsg) > 0 {
		tok := expr.Right.GetToken()
		tok.Text = expr.Right.String()
		s.collector.Add(s.collector.Error(tok, errMsg))
	}

	switch expr.Right.(type) {
	case *ast.IfExpression:
		errMsg = "ERROR: can't have an if expression as the right side of a binary expression"
	case *ast.ArrayLiteral:
		errMsg = "ERROR: can't have an array literal as the right side of a binary expression"
	case *ast.MapLiteral:
		errMsg = "ERROR: can't have a map literal as the right side of a binary expression"
	default:
		s.symbolReaderExpression(expr.Right)
	}

	if len(errMsg) > 0 {
		tok := expr.Right.GetToken()
		tok.Text = expr.Right.String()
		s.collector.Add(s.collector.Error(tok, errMsg))
	}
}

func (s *TypeChecker) visitIfExpression(expr *ast.IfExpression) {
	conditionExpression := expr.Condition

	errMsg := ""
	switch cExpr := conditionExpression.(type) {
	case *ast.ArrayLiteral:
		errMsg = "ERROR: can't use an array literal as condition in if expression"
	case *ast.MapLiteral:
		errMsg = "ERROR: can't use a map literal as condition in if expression"
	default:
		// rest of allowed operation
		s.symbolReaderExpression(cExpr)
	}

	if len(errMsg) > 0 {
		tok := conditionExpression.GetToken()
		tok.Text = conditionExpression.String()
		s.collector.Add(s.collector.Error(tok, errMsg))
	}

	// s.inference.inferAssociatedValueType(conditionExpression)

	// check the body of if
	s.visitBlockDCL(expr.Consequence)

	// check the body of the else, if it exists
	switch alt := expr.Alternative.(type) {
	case *ast.BlockStatement:
		s.visitBlockDCL(alt)
	case *ast.IfExpression:
		s.visitIfExpression(alt)
	default:
		// Do nothing
	}
}

func (s *TypeChecker) visitIdentifier(expr *ast.Identifier) *SymbolInfo {
	// if (identifier) check if it declared or not
	ident, isMatched := s.symbols.Resolve(expr.Value)

	if !isMatched {
		errMsg := ("ERROR: identifier, needs to be declared before it gets used")
		s.collector.Add(s.collector.Error(expr.Token, errMsg))
		return nil
	}

	return ident
}

func (s *TypeChecker) visitStructInstanceExpression(expr *ast.StructInstanceExpression) {

	identifier, ok := expr.Left.(*ast.Identifier)

	if ok {
		structDef, isMatched := s.symbols.Resolve(identifier.Value)

		if !isMatched {
			errMsg := fmt.Sprintf("ERROR: struct (%v) needs to be defined, before creating instances of it", identifier.Value)
			s.collector.Add(s.collector.Error(identifier.Token, errMsg))
			return
		}

		body := expr.Body

		keys := &ast.StructExpression{}

		switch strcDf := structDef.DeclNode.(*ast.VarDeclaration).Value.(type) {
		case *ast.StructExpression:
			keys = strcDf
			// case *ast.TypeStatement:
			// 	keys = s.visitIdentifier(
			// 		&ast.Identifier{
			// 			Value: strcDf.Value.(*ast.NodeType).Type,
			// 		},
			// 	).DeclNode.(*ast.StructStatement)
		}

		// means that some fields are left out of the having an associated value
		for _, field := range keys.Body {
			// find the key in the struct instance
			idx := slices.IndexFunc(body, func(f ast.FieldInstance) bool {
				return f.Key.Value == field.Key.Value
			})

			// the field only gets initialized if it a valid type and not a build in method (function)

			_, isFieldMethod := field.Value.(*ast.FunctionExpression)

			if idx == -1 && !isFieldMethod {
				errMsg := fmt.Sprintf("ERROR: field (%v) needs to be instantiated with a value, cause it exists on the struct definition", field.Key.Value)
				s.collector.Add(s.collector.Error(field.Key.Token, errMsg))
			}

			if idx != -1 && isFieldMethod {
				errMsg := fmt.Sprintf("ERROR: field (%v) is a builtin method into (%v) struct, u can't override it consider rewriting the definition of it into an a valid type", field.Key.Value, structDef.Name)
				s.collector.Add(s.collector.Error(field.Key.Token, errMsg))
			}
		}

		for _, field := range body {
			// find the key in the struct instance
			idx := slices.IndexFunc(keys.Body, func(f ast.Field) bool {
				return f.Key.Value == field.Key.Value
			})

			if idx == -1 {
				errMsg := fmt.Sprintf("ERROR: field (%v) doesn't exist on the struct definition, either add it to the definition or remove the field from the instance", field.Key.Value)
				s.collector.Add(s.collector.Error(field.Key.Token, errMsg))
			}
		}
	}
}

func (s *TypeChecker) visitArrayLiteral(expr *ast.ArrayLiteral) {
	elements := expr.Elements

	for _, elem := range elements {
		switch elem.(type) {
		case *ast.IfExpression:
			errMsg := "ERROR: can't use an if expression as value in array literal"
			tok := elem.GetToken()
			tok.Text = elem.String()
			s.collector.Add(s.collector.Error(tok, errMsg))
		default:
			s.symbolReaderExpression(elem)
		}
	}
}

func (s *TypeChecker) visitMapLiteral(expr *ast.MapLiteral) {
	pairs := expr.Pairs
	errMsg := ""
	for key, value := range pairs {
		switch k := key.(type) {
		case *ast.IfExpression:
			errMsg = "ERROR: can't use an if expression as key in a map literal"
		case *ast.MapLiteral:
			errMsg = "ERROR: can't use an map literal as key of a map literal"
		case *ast.ArrayLiteral:
			errMsg = "ERROR: can't use an array literal as key in map literal"
		default:
			s.symbolReaderExpression(k)
		}
		if len(errMsg) > 0 {
			tok := key.GetToken()
			tok.Text = key.String()
			s.collector.Add(s.collector.Error(tok, errMsg))
		}
		switch v := value.(type) {
		case *ast.IfExpression:
			errMsg = "ERROR: can't use an if expression as value in a map literal"
			tok := key.GetToken()
			tok.Text = key.String()
			s.collector.Add(s.collector.Error(tok, errMsg))
		default:
			s.symbolReaderExpression(v)
		}
	}
}

func (s *TypeChecker) visitIndexExpression(expr *ast.IndexExpression) {
	// check if the lest side is a valid
	errMsg := ""
	switch lf := expr.Left.(type) {
	case *ast.IfExpression:
		errMsg = "ERROR: can't use an if expression as left side of index expression"
	case *ast.BinaryExpression:
		errMsg = "ERROR: can't use a binary expression as left side of index expression, cause it evaluates to a boolean"
		// construct the token
	case *ast.UnaryExpression:
		errMsg = "ERROR: can't use a unary expression as left side of index expression, cause it evaluates to a boolean"
	case *ast.StructInstanceExpression:
		errMsg = "ERROR: can't use a struct instance as left side of index expression"
	default:
		// nothing
		s.symbolReaderExpression(lf)
	}

	if len(errMsg) > 0 {
		tok := expr.Left.GetToken()
		tok.Text = expr.Left.String()
		s.collector.Add(s.collector.Error(tok, errMsg))
	}

	switch rf := expr.Index.(type) {
	case *ast.BinaryExpression:
		errMsg = "ERROR: can't use a binary expression, cause it evaluates to a boolean"
	case *ast.UnaryExpression:
		errMsg = "ERROR: can't use a unary expression, cause it evaluates to a boolean"
	case *ast.IfExpression:
		errMsg = "ERROR: can't use an if expression as index, index can only be an int"
	default:
		s.symbolReaderExpression(rf)
	}

	if len(errMsg) > 0 {
		tok := expr.Left.GetToken()
		tok.Text = expr.Left.String()
		s.collector.Add(s.collector.Error(tok, errMsg))
	}
}

func (s *TypeChecker) visitMemberShipAccess(expr *ast.MemberShipExpression) {
	switch or := expr.Object.(type) {
	case *ast.Identifier:
		s.visitIdentifier(or)
	case *ast.MemberShipExpression:
		s.visitMemberShipAccess(or)
	default:
		errMsg := fmt.Sprintf("ERROR: can't use %v as the main object", expr.Object)
		tok := expr.Object.GetToken()
		tok.Text = expr.Object.String()
		s.collector.Add(s.collector.Error(tok, errMsg))
	}

	switch expr.Property.(type) {
	case *ast.Identifier, *ast.CallExpression:
		// no need for checks, since the type inference will error if not found
	default:
		// everything here is refused
		errMsg := fmt.Sprintf("ERROR: can't use %v as the property, and access it", expr.Property)
		tok := expr.Object.GetToken()
		tok.Text = expr.Object.String()
		s.collector.Add(s.collector.Error(tok, errMsg))
	}
}
