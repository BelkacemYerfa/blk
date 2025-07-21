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
		errMsg := fmt.Sprintf("ERROR: %v identifier is already declared", sym.Name)
		s.Collector.Add(s.Error(node.Token, errMsg))
	}

	if len(node.Body) > 0 {
		// check fields
		nwTab := NewSymbolTable(s.Tokens, s.Collector)
		nwTab.Parent = s
		nwTab.DepthIndicator++
		for _, field := range node.Body {
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
		tok := node.Name.Token
		errMsg := fmt.Sprintf("ERROR: %v identifier is already declared", node.Name.Value)
		s.Collector.Add(s.Error(tok, errMsg))
	} else {
		s.Define(sym.Name, sym)
	}
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

	_, isMatched := s.Resolve(functionName)

	if !isMatched {
		errMsg := fmt.Sprintf("ERROR: (%v) function, needs to be declared before it get called", expr)
		s.Collector.Add(s.Error(expr.Token, errMsg))
	}

	// check if same number of the args provided is the same
	args := expr.Args
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
