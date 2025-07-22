package compiler

import (
	"blk/internals"
	"blk/parser"
	"errors"
	"fmt"
	"sort"
	"strings"
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
	Collector      internals.ErrorCollector
	Tokens         []parser.Token
}

func NewSymbolTable(tokens []parser.Token, errCollector internals.ErrorCollector) *SymbolTable {
	return &SymbolTable{
		Store:     make(map[string]SymbolInfo),
		Tokens:    tokens,
		Collector: errCollector,
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

func (s *SymbolTable) Error(tok parser.Token, msg string) error {
	errMsg := fmt.Sprintf("\033[1;90m%s:%d:%d:\033[0m\n\n", "main.blk", tok.Row, tok.Col)

	// Build row set map
	rowSet := make(map[int][]parser.Token)
	for _, t := range s.Tokens {
		rowSet[t.Row] = append(rowSet[t.Row], t)
	}

	// Collect sorted rows
	rows := []int{}
	for row := range rowSet {
		rows = append(rows, row)
	}
	sort.Ints(rows)

	// Find closest previous and next row
	var prevRow, nextRow int
	prevRow, nextRow = -1, -1
	for _, row := range rows {
		if row < tok.Row {
			prevRow = row
		} else if row > tok.Row && nextRow == -1 {
			nextRow = row
		}
	}

	// Build rowMap with only prevRow, tok.Row, nextRow
	rowMap := make(map[int][]parser.Token)
	if prevRow != -1 {
		rowMap[prevRow] = rowSet[prevRow]
	}
	rowMap[tok.Row] = rowSet[tok.Row]
	if nextRow != -1 {
		rowMap[nextRow] = rowSet[nextRow]
	}

	// Format rows
	formattedRows := []int{}
	for row := range rowMap {
		formattedRows = append(formattedRows, row)
	}
	sort.Ints(formattedRows)

	for _, row := range formattedRows {
		currentLine := rowMap[row]
		lineContent := ""
		lastCol := 0

		for _, t := range currentLine {
			if t.Col > lastCol {
				lineContent += strings.Repeat(" ", t.Col-lastCol)
			}
			if t.Kind == parser.TokenString {
				t.Text = fmt.Sprintf(`"%s"`, t.Text)
			}
			lineContent += t.Text
			lastCol = t.Col + len(t.Text)
		}

		lineNumStr := fmt.Sprintf("%d", row)
		errMsg += fmt.Sprintf("%s    %s\n", lineNumStr, lineContent)

		if row == tok.Row {
			spacesBeforeLineNum := len(lineNumStr)
			spacesAfterLineNum := 4
			spacesBeforeToken := tok.Col

			totalSpaces := spacesBeforeLineNum + spacesAfterLineNum + spacesBeforeToken

			errorIndicator := strings.Repeat(" ", totalSpaces)
			errMsg += errorIndicator + "\033[1;31m"
			repeat := len(tok.Text)
			if repeat == 0 {
				repeat = 1
			}
			errMsg += strings.Repeat("^", repeat)
			errMsg += "\033[0m\n"
		}
	}

	errMsg += msg
	// skip until the next useful line

	return errors.New(errMsg)
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
	case *parser.ScopeStatement:
		s.visitScopeDCL(node)
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
		errMsg := fmt.Sprintf("ERROR: %v identifier is already declared", sym.Name)
		s.Collector.Add(s.Error(node.Token, errMsg))
	}

	if node.Body != nil {
		s.visitBlockDCL(node.Body)
	}

	// checks if the arg type is declared or not
	for _, arg := range node.Args {
		argType := arg.Type
		s.visitFieldType(argType)
	}

	s.visitFieldType(node.ReturnType)
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
		errMsg := fmt.Sprintf("ERROR: %v identifier is already declared", sym.Name)
		s.Collector.Add(s.Error(node.Token, errMsg))
	}

	s.visitFieldType(node.ExplicitType)
	s.Define(sym.Name, sym)
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
		errMsg := fmt.Sprintf("ERROR: %v identifier is already declared", sym.Name)
		s.Collector.Add(s.Error(node.Token, errMsg))
	}

	s.Define(sym.Name, sym)

	if len(node.Body) > 0 {
		nwTab := NewSymbolTable(s.Tokens, s.Collector)
		nwTab.Parent = s
		nwTab.DepthIndicator++
		for _, field := range node.Body {
			// check for field redundancy
			fieldName := field.Key.Value
			_, ok := nwTab.Resolve(fieldName)
			if ok {
				errMsg := fmt.Sprintf("ERROR: ( %v ) key is already declared, attempt to re-declare", fieldName)
				s.Collector.Add(s.Error(field.Key.Token, errMsg))
				return
			} else {
				nwTab.Define(fieldName, &SymbolInfo{
					Name:     field.Key.Value,
					Kind:     SymbolIdentifier,
					Depth:    nwTab.DepthIndicator,
					DeclNode: field.Key,
				})
			}

			// check for if the type is custom type
			// and if it is check if it already in the table or not
			fieldType := field.Value
			s.visitFieldType(fieldType)
		}
	}

}

func (s *SymbolTable) visitFieldType(fieldType parser.Expression) {

	switch tp := fieldType.(type) {
	case *parser.NodeType:
		if _, ok := parser.AtomicTypes[tp.Type]; !ok {
			if tp.Type != "array" {
				_, exist := s.Resolve(tp.Type)
				if !exist {
					errMsg := fmt.Sprintf("ERROR: type ( %v ) needs to be declared before it gets used", tp.Type)
					s.Collector.Add(s.Error(tp.Token, errMsg))
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
				_, exist := s.Resolve(tp.Type)
				if !exist {
					errMsg := fmt.Sprintf("ERROR: type ( %v ) needs to be declared before it gets used", tp.Type)
					s.Collector.Add(s.Error(tp.Token, errMsg))
				}
			}
		}

		// check for this in a recursive manner if the tp.(Left | Right) != nil
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

func (s *SymbolTable) visitTypeDCL(node *parser.TypeStatement) {
	sym := &SymbolInfo{
		Name:     node.Name.Value,
		Kind:     SymbolType,
		Depth:    s.DepthIndicator,
		DeclNode: node,
	}

	_, ok := s.Resolve(sym.Name)

	if ok {
		tok := node.Name.Token
		errMsg := fmt.Sprintf("ERROR: %v identifier is already declared", node.Name.Value)
		s.Collector.Add(s.Error(tok, errMsg))
	}

	s.visitFieldType(node.Value)
	s.Define(sym.Name, sym)
}

func (s *SymbolTable) visitBlockDCL(block *parser.BlockStatement) {
	if len(block.Body) > 0 {
		nwTab := NewSymbolTable(s.Tokens, s.Collector)
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
		errMsg := "ERROR: return statement, can't be on the global scope, needs to be inside of a function"
		s.Collector.Add(s.Error(node.Token, errMsg))
	}

	identifier, ok := node.ReturnValue.(*parser.Identifier)

	if ok {
		// if (identifier) check if it declared or not
		_, isMatched := s.Resolve(identifier.Value)

		if !isMatched {
			errMsg := ("ERROR: identifier, needs to be declared before it gets returned")
			s.Collector.Add(s.Error(node.Token, errMsg))
		}
	}
}

func (s *SymbolTable) visitScopeDCL(node *parser.ScopeStatement) {
	if node.Body != nil {
		s.visitBlockDCL(node.Body)
	}
}

func (s *SymbolTable) visitCallExpression(expr *parser.CallExpression) {
	functionName := expr.Function.Value

	function, isMatched := s.Resolve(functionName)

	if !isMatched {
		errMsg := fmt.Sprintf("ERROR: (%v) function, needs to be declared before it get called", expr)
		s.Collector.Add(s.Error(expr.Token, errMsg))
		return
	}

	// check if same number of the args provided is the same
	args := expr.Args

	dclNode := function.DeclNode.(*parser.FunctionStatement)
	if len(args) < len(dclNode.Args) {
		errMsg := "ERROR: need to pass all the args into the function call"
		tok := dclNode.Args[len(args)].Token
		expr.Token.Col = expr.Token.Col + len(expr.Token.Text)
		expr.Token.Text = tok.Text
		s.Collector.Add(s.Error(expr.Token, errMsg))
		return
	}

	if len(args) > len(dclNode.Args) {
		errMsg := "ERROR: function call is receiving more args than it should"
		tok := parser.Token{}
		startIdx := len(dclNode.Args)

		for idx, arg := range args[startIdx:] {
			switch expr := arg.(type) {
			case *parser.CallExpression:
				tok.Text += expr.String()
				tok.Row = expr.Token.Row
				if idx == 0 {
					tok.Col = expr.Token.Col
				}
			case *parser.MemberShipExpression:
				tok.Text += expr.String()
				tok.Row = expr.Token.Row
				if idx == 0 {
					tok.Col = expr.Token.Col
				}

			case *parser.ArrayLiteral:
				tok.Text += expr.String()
				tok.Row = expr.Token.Row
				if idx == 0 {
					tok.Col = expr.Token.Col
				}

			case *parser.MapLiteral:
				tok.Text += expr.String()
				tok.Row = expr.Token.Row
				if idx == 0 {
					tok.Col = expr.Token.Col
				}

			case *parser.BooleanLiteral:
				tok.Text += expr.String()
				tok.Row = expr.Token.Row
				if idx == 0 {
					tok.Col = expr.Token.Col
				}

			case *parser.StringLiteral:
				tok.Text += expr.String()
				tok.Row = expr.Token.Row
				if idx == 0 {
					tok.Col = expr.Token.Col
				}

			case *parser.FloatLiteral:
				tok.Text += expr.String()
				tok.Row = expr.Token.Row
				if idx == 0 {
					tok.Col = expr.Token.Col
				}

			case *parser.IntegerLiteral:
				tok.Text += expr.String()
				tok.Row = expr.Token.Row
				if idx == 0 {
					tok.Col = expr.Token.Col
				}

			case *parser.Identifier:
				tok.Text += expr.String()
				tok.Row = expr.Token.Row
				if idx == 0 {
					tok.Col = expr.Token.Col
				}

			case *parser.BinaryExpression:
				tok.Text += expr.String()
				tok.Row = expr.Token.Row
				if idx == 0 {
					tok.Col = expr.Token.Col
				}

			case *parser.UnaryExpression:
				tok.Text += expr.String()
				tok.Row = expr.Token.Row
				if idx == 0 {
					tok.Col = expr.Token.Col
				}

			default:
			}
			if idx+1 <= len(args)-startIdx-1 {
				tok.Text += ", "
			}
		}
		fmt.Println(tok.Text)
		s.Collector.Add(s.Error(tok, errMsg))
		return
	}

	// check if the args of the call expr, if they already exist

	for _, arg := range args {
		identifier, ok := arg.(*parser.Identifier)
		if ok {
			_, isMatched := s.Resolve(identifier.Value)

			if !isMatched {
				errMsg := "ERROR: identifier, needs to be declared before it gets used in a function call"
				s.Collector.Add(s.Error(identifier.Token, errMsg))
			}
		}
	}
}

func (s *SymbolTable) visitUnaryExpression(expr *parser.UnaryExpression) {
	identifier, ok := expr.Right.(*parser.Identifier)

	if ok {
		// if (identifier) check if it declared or not
		_, isMatched := s.Resolve(identifier.Value)

		if !isMatched {
			errMsg := ("ERROR: identifier, needs to be declared before it gets checked")
			s.Collector.Add(s.Error(expr.Token, errMsg))
		}
	}
}

func (s *SymbolTable) visitBinaryExpression(expr *parser.BinaryExpression) {

	switch lExpr := expr.Left.(type) {
	case *parser.Identifier:
		_, isMatched := s.Resolve(lExpr.Value)

		if !isMatched {
			errMsg := fmt.Sprintf("ERROR: (%v) left identifier, needs to be declared before it gets checked", lExpr.Value)
			s.Collector.Add(s.Error(lExpr.Token, errMsg))
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
			errMsg := fmt.Sprintf("ERROR: (%v) right identifier, needs to be declared before it gets checked", rExpr.Value)
			s.Collector.Add(s.Error(expr.Token, errMsg))
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
