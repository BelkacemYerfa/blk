package compiler

import (
	"blk/parser"
	"fmt"
)

type SymbolKind = string

const (
	SymbolVar        SymbolKind = "var"
	SymbolLet        SymbolKind = "let"
	SymbolFunc       SymbolKind = "fn"
	SymbolFor        SymbolKind = "for"
	SymbolWhile      SymbolKind = "while"
	SymbolStruct     SymbolKind = "struct"
	SymbolType       SymbolKind = "type"
	SymbolIdentifier SymbolKind = "identifier"
)

type SymbolInfo struct {
	Name      string
	DeclNode  any        // pointer to node dcl in AST
	Kind      SymbolKind // func, var, param, let...
	IsMutable bool
	Depth     int
}

type SymbolTable struct {
	Parent         *SymbolTable          // for nested scopes
	Store          map[string]SymbolInfo // current scope's entries
	DepthIndicator int
	Errors         []error
}

func NewSymbolTable() *SymbolTable {
	return &SymbolTable{
		Store: make(map[string]SymbolInfo),
	}
}

func (s *SymbolTable) Define(name string, sym *SymbolInfo) {
	s.Store[name] = *sym
}

func (s *SymbolTable) Resolve(name string) (*SymbolInfo, bool) {
	curr := s
	if sym, ok := curr.Store[name]; ok {
		return &sym, true
	}
	if curr.Parent != nil {
		sym, _ := curr.Parent.Resolve(name)
		if sym != nil {
			if sym.Depth == s.DepthIndicator {
				return sym, true
			}
		}
	}
	return nil, false
}

func (s *SymbolTable) SymbolBuilder(ast *parser.Program) {
	for _, node := range ast.Statements {
		s.symbolReader(node)
	}
}

func (s *SymbolTable) symbolReader(node parser.Statement) {
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
		s.visitBlockDCL(node.Body)
	case *parser.ForStatement:
		s.visitBlockDCL(node.Body)
	case *parser.ReturnStatement:
		s.visitReturnDCL(node)
	default:
		// type is an expression statement
		stmt := node.(*parser.ExpressionStatement)
		s.symbolReaderExpression(stmt)
	}
}

func (s *SymbolTable) symbolReaderExpression(node *parser.ExpressionStatement) {
	switch expr := node.Expression.(type) {
	case *parser.CallExpression:
		s.visitCallExpression(expr)
	case *parser.UnaryExpression:
		s.visitUnaryExpression(expr)
	case *parser.IfExpression:
		s.visitIfExpression(expr)
	case *parser.BinaryExpression:
		s.visitBinaryExpression(expr)
	default:

	}
}

func (s *SymbolTable) visitFuncDCL(node *parser.FunctionStatement) {
	sym := &SymbolInfo{
		Name:     node.Name,
		Kind:     SymbolFunc,
		Depth:    s.DepthIndicator,
		DeclNode: node,
	}

	_, ok := s.Resolve(sym.Name)

	if ok {
		errMsg := fmt.Errorf("ERROR: %v identifier is already declared", sym.Name)
		fmt.Println(errMsg)
		return
	}

	if node.Body != nil {
		s.visitBlockDCL(node.Body)
	}

	s.Define(sym.Name, sym)
}

func (s *SymbolTable) visitVarDCL(node *parser.LetStatement) {
	kind := SymbolLet
	isMutable := false
	if node.Token.Text == "var" {
		kind = SymbolVar
		isMutable = true
	}

	sym := &SymbolInfo{
		Name:      node.Name.Value,
		Kind:      kind,
		Depth:     s.DepthIndicator,
		IsMutable: isMutable,
		DeclNode:  node,
	}

	_, ok := s.Resolve(sym.Name)

	if ok {
		errMsg := fmt.Errorf("ERROR: %v identifier is already declared", sym.Name)
		fmt.Println(errMsg)
		return
	} else {
		s.Define(sym.Name, sym)
	}
}

func (s *SymbolTable) visitStructDCL(node *parser.StructStatement) {

	sym := &SymbolInfo{
		Name:     node.Name.Value,
		Kind:     SymbolStruct,
		Depth:    s.DepthIndicator,
		DeclNode: node,
	}

	_, ok := s.Resolve(sym.Name)

	if ok {
		errMsg := fmt.Errorf("ERROR: %v identifier is already declared", sym.Name)
		fmt.Println(errMsg)
		return
	}

	if len(node.Body) > 0 {
		// check fields
		for _, field := range node.Body {
			fieldName := field.Key.Value
			_, ok := s.Resolve(fieldName)
			if ok {
				errMsg := fmt.Errorf("ERROR: ( %v ) key is already declared, attempt to re-declare", fieldName)
				fmt.Println(errMsg)
				return
			} else {
				s.Define(fieldName, &SymbolInfo{
					Name:     node.Name.Value,
					Kind:     SymbolIdentifier,
					Depth:    s.DepthIndicator + 1,
					DeclNode: node,
				})
			}
		}
	}

	s.Define(sym.Name, sym)
}

func (s *SymbolTable) visitTypeDCL(node *parser.TypeStatement) {
	sym := &SymbolInfo{
		Name:     node.Name.Value,
		Kind:     SymbolType,
		Depth:    s.DepthIndicator,
		DeclNode: node,
	}

	_, ok := s.Resolve(sym.Name)

	if ok {
		errMsg := fmt.Errorf("ERROR: %v identifier is already declared", sym.Name)
		fmt.Println(errMsg)
		return
	} else {
		s.Define(sym.Name, sym)
	}
}

func (s *SymbolTable) visitBlockDCL(block *parser.BlockStatement) {
	if len(block.Body) > 0 {
		nwTab := NewSymbolTable()
		nwTab.Parent = s
		nwTab.DepthIndicator = s.DepthIndicator + 1
		for _, nd := range block.Body {
			nwTab.symbolReader(nd)
		}
	}
}

func (s *SymbolTable) visitReturnDCL(node *parser.ReturnStatement) {
	if s.DepthIndicator == 0 {
		// means it is on the global scope not in a function
		errMsg := fmt.Errorf("ERROR: return statement, can't be on the global scope, needs to be inside of a function")
		fmt.Println(errMsg)
		return
	}

	identifier, ok := node.ReturnValue.(*parser.Identifier)

	if ok {
		// if (identifier) check if it declared or not
		_, isMatched := s.Resolve(identifier.Value)

		if !isMatched {
			errMsg := fmt.Errorf("ERROR: identifier, needs to be declared before it gets returned")
			fmt.Println(errMsg)
			return
		}
	}

}

func (s *SymbolTable) visitCallExpression(expr *parser.CallExpression) {
	functionName := expr.Function.Value

	_, isMatched := s.Resolve(functionName)

	if !isMatched {
		errMsg := fmt.Errorf("ERROR: (%v) function, needs to be declared before it get called", expr)
		fmt.Println(errMsg)
		return
	}
}

func (s *SymbolTable) visitUnaryExpression(expr *parser.UnaryExpression) {
	identifier, ok := expr.Right.(*parser.Identifier)

	if ok {
		// if (identifier) check if it declared or not
		_, isMatched := s.Resolve(identifier.Value)

		if !isMatched {
			errMsg := fmt.Errorf("ERROR: identifier, needs to be declared before it gets checked")
			fmt.Println(errMsg)
			return
		}
	}
}

func (s *SymbolTable) visitBinaryExpression(expr *parser.BinaryExpression) {

	switch lExpr := expr.Left.(type) {
	case *parser.Identifier:
		_, isMatched := s.Resolve(lExpr.Value)

		if !isMatched {
			errMsg := fmt.Errorf("ERROR: (%v) left identifier, needs to be declared before it gets checked", lExpr.Value)
			fmt.Println(errMsg)
			return
		}
	case *parser.BinaryExpression:
		s.visitBinaryExpression(lExpr)
	case *parser.CallExpression:
		s.visitCallExpression(lExpr)
	default:

	}

	switch rExpr := expr.Right.(type) {
	case *parser.Identifier:
		_, isMatched := s.Resolve(rExpr.Value)

		if !isMatched {
			errMsg := fmt.Errorf("ERROR: (%v) right identifier, needs to be declared before it gets checked", rExpr.Value)
			fmt.Println(errMsg)
			return
		}
	case *parser.BinaryExpression:
		s.visitBinaryExpression(rExpr)
	case *parser.CallExpression:
		s.visitCallExpression(rExpr)
	default:

	}

}

func (s *SymbolTable) visitIfExpression(expr *parser.IfExpression) {
	conditionExpression := expr.Condition

	switch cExpr := conditionExpression.(type) {
	case *parser.UnaryExpression:
		s.visitUnaryExpression(cExpr)
	case *parser.BinaryExpression:
		s.visitBinaryExpression(cExpr)
	default:
		//
	}
}
