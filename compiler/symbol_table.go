package compiler

import (
	"blk/internals"
	"blk/parser"
	"errors"
	"fmt"
	"os"
	"slices"
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
	Type      any
	IsMutable bool
	Depth     int
}

type SymbolTable struct {
	Parent           *SymbolTable          // for nested scopes
	Store            map[string]SymbolInfo // current scope's entries
	EmbeddedSymTable []*SymbolTable
	DepthIndicator   int
	CurrNode         *SymbolInfo
}

func NewSymbolTable() *SymbolTable {
	return &SymbolTable{
		Store:            make(map[string]SymbolInfo),
		EmbeddedSymTable: make([]*SymbolTable, 0),
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
			return sym, true
		}
	}
	return nil, false
}

type TypeChecker struct {
	Symbols   *SymbolTable
	Collector internals.ErrorCollector
	Tokens    []parser.Token
}

func NewTypeChecker(tokens []parser.Token, errCollector internals.ErrorCollector) *TypeChecker {
	return &TypeChecker{
		Tokens:    tokens,
		Symbols:   NewSymbolTable(),
		Collector: errCollector,
	}
}

func (s *TypeChecker) Error(tok parser.Token, msg string) error {
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
	return errors.New(errMsg)
}

func (s *TypeChecker) insertUniqueErrors(errMsg error) {
	_, found := slices.BinarySearchFunc(s.Collector.Errors, errMsg, func(a, b error) int {
		return strings.Compare(a.Error(), b.Error())
	})
	if !found {
		s.Collector.Add(errMsg)
	}
}

func (s *TypeChecker) SymbolBuilder(ast *parser.Program) {

	// insert prebuilt func later on

	for _, node := range ast.Statements {
		s.symbolReader(node)
	}

	// check if the entry point is a main function
	// access to the main
	mainFn := s.Symbols.Store["main"]
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
		Depth:    s.Symbols.DepthIndicator,
		DeclNode: node,
		Type:     node.ReturnType,
	}

	// check if the name doesn't collide with pre-built function
	if _, isMatching := builtInFunctions[sym.Name]; isMatching {
		errMsg := fmt.Sprintf("ERROR: fn %v is a pre-built function, consider renaming your function to something else", sym.Name)
		s.Collector.Add(s.Error(node.Name.Token, errMsg))
	}

	_, ok := s.Symbols.Resolve(sym.Name)

	if ok {
		errMsg := fmt.Sprintf("ERROR:fn %v is already declared, consider removing the duplicate", sym.Name)
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
	s.Symbols.Define(sym.Name, sym)
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
		Depth:     s.Symbols.DepthIndicator,
		IsMutable: isMutable,
		DeclNode:  node,
		Type:      node.ExplicitType,
	}

	_, ok := s.Symbols.Resolve(sym.Name)
	if ok {
		errMsg := fmt.Sprintf("ERROR: %v identifier is already declared", sym.Name)
		s.Collector.Add(s.Error(node.Token, errMsg))
	}

	s.visitFieldType(node.ExplicitType)
	s.Symbols.CurrNode = sym
	s.symbolReaderExpression(node.Value)
	expectedType := node.ExplicitType.String()
	gotType := s.inferAssociatedValueType(node.Value)

	if expectedType != gotType.String() {
		errMsg := ""
		if gotType.String() == "nil" {
			errMsg = fmt.Sprintf("ERROR: type mismatch, expected %v, got %v, check nest level of the array", expectedType, gotType)
		} else {
			errMsg = fmt.Sprintf("ERROR: type mismatch, expected %v, got %v", expectedType, gotType)
		}
		tok := gotType.GetToken()
		if gotType.GetType() == parser.TokenMap {
			tok.Text = node.ExplicitType.String()
			tok.Col = node.ExplicitType.GetToken().Col - len(tok.Text) - 1
		}
		s.Collector.Add(s.Error(tok, errMsg))
	}

	s.Symbols.Define(sym.Name, sym)
}

func (s *TypeChecker) visitStructDCL(node *parser.StructStatement) *SymbolInfo {

	sym := &SymbolInfo{
		Name:     node.Name.Value,
		Kind:     SymbolStruct,
		Depth:    s.Symbols.DepthIndicator,
		DeclNode: node,
	}

	_, ok := s.Symbols.Resolve(sym.Name)

	if ok {
		errMsg := fmt.Sprintf("ERROR: %v identifier is already declared", sym.Name)
		s.Collector.Add(s.Error(node.Token, errMsg))
	}

	s.Symbols.Define(sym.Name, sym)

	if len(node.Body) > 0 {
		nwTab := NewSymbolTable()
		s.Symbols.EmbeddedSymTable = append(s.Symbols.EmbeddedSymTable, nwTab)
		nwTab.Parent = s.Symbols
		nwTab.DepthIndicator++
		for _, field := range node.Body {
			// check for field redundancy
			fieldName := field.Key.Value
			_, ok := nwTab.Resolve(fieldName)
			if ok {
				errMsg := fmt.Sprintf("ERROR: ( %v ) key is already declared, attempt to re-declare", fieldName)
				s.Collector.Add(s.Error(field.Key.Token, errMsg))
				return nil
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

	return sym
}

func (s *TypeChecker) visitFieldType(fieldType parser.Expression) {

	switch tp := fieldType.(type) {
	case *parser.NodeType:
		if _, ok := parser.AtomicTypes[tp.Type]; !ok {
			if tp.Type != "array" {
				_, exist := s.Symbols.Resolve(tp.Type)
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
				_, exist := s.Symbols.Resolve(tp.Type)
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

func (s *TypeChecker) visitTypeDCL(node *parser.TypeStatement) {
	sym := &SymbolInfo{
		Name:     node.Name.Value,
		Kind:     SymbolType,
		Depth:    s.Symbols.DepthIndicator,
		DeclNode: node,
	}

	_, ok := s.Symbols.Resolve(sym.Name)

	if ok {
		tok := node.Name.Token
		errMsg := fmt.Sprintf("ERROR: %v identifier is already declared", node.Name.Value)
		s.Collector.Add(s.Error(tok, errMsg))
	}

	s.visitFieldType(node.Value)
	s.Symbols.Define(sym.Name, sym)
}

func (s *TypeChecker) visitBlockDCL(block *parser.BlockStatement) {
	if len(block.Body) > 0 {
		nwTab := NewSymbolTable()
		nwTab.Parent = s.Symbols
		nwTab.DepthIndicator++
		s.Symbols = nwTab
		for _, nd := range block.Body {
			s.symbolReader(nd)
		}
		s.Symbols = s.Symbols.Parent
	}
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
			Depth:    s.Symbols.DepthIndicator,
		}

		s.Symbols.Define(sym.Name, sym)
	}

	// check for the target if it already existing
	s.symbolReaderExpression(node.Target)
	// check for the body
	s.visitBlockDCL(node.Body)
}

func (s *TypeChecker) visitReturnDCL(node *parser.ReturnStatement) {
	if s.Symbols.DepthIndicator == 0 {
		// means it is on the global scope not in a function
		errMsg := "ERROR: return statement, can't be on the global scope, needs to be inside of a function"
		s.Collector.Add(s.Error(node.Token, errMsg))
	}

	identifier, ok := node.ReturnValue.(*parser.Identifier)

	if ok {
		// if (identifier) check if it declared or not
		s.visitIdentifier(identifier)
	}
}

func (s *TypeChecker) visitScopeDCL(node *parser.ScopeStatement) {
	if node.Body != nil {
		s.visitBlockDCL(node.Body)
	}
}

func (s *TypeChecker) visitCallExpression(expr *parser.CallExpression) {
	functionName := expr.Function.Value

	function, isMatched := s.Symbols.Resolve(functionName)

	_, isBuiltInFunc := builtInFunctions[functionName]

	if !isMatched && !isBuiltInFunc {
		errMsg := fmt.Sprintf("ERROR: (%v) function, needs to be declared before it get called", expr.Function.Value)
		s.Collector.Add(s.Error(expr.Token, errMsg))
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
		s.Collector.Add(s.Error(expr.Token, errMsg))
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
		s.Collector.Add(s.Error(tok, errMsg))
		return
	}

	// check if the args of the call expr, already exist
	// TODO: check also if there associated type is the same as the type of args on the function signature
	for idx, arg := range args {
		switch ag := arg.(type) {
		case *parser.Identifier:
			s.visitIdentifier(ag)
			returnType := s.inferAssociatedValueType(ag)
			tp := dclNode.Args[idx].Type
			s.argTypeChecker(tp.String(), returnType.String(), arg)
		case *parser.MemberShipExpression:
			s.visitMemberShipAccess(ag)

		default:
			// handle only atomic types (strings, booleans, floats, ints)
			returnType := s.inferAssociatedValueType(ag)
			tp := dclNode.Args[idx].Type
			s.argTypeChecker(tp.String(), returnType.String(), arg)
		}
	}
}

func (s *TypeChecker) argTypeChecker(tp, returnType string, arg parser.Expression) {
	if tp != returnType {
		errMsg := fmt.Sprintf("ERROR: type mismatch, expected %v, got %v", tp, returnType)
		tok := arg.GetToken()
		tok.Text = arg.String()
		s.Collector.Add(s.Error(tok, errMsg))
	}
}

func (s *TypeChecker) visitUnaryExpression(expr *parser.UnaryExpression) {
	// check if the type is boolean or not
	if s.Symbols.CurrNode.Type.(*parser.NodeType).Type == parser.BoolType && expr.Operator == "-" {
		errMsg := "ERROR: can't use operator (-) with boolean types, only operator (!) is allowed"
		s.Collector.Add(s.Error(expr.Token, errMsg))
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
	ident, isMatched := s.Symbols.Resolve(expr.Value)

	if !isMatched {
		errMsg := ("ERROR: identifier, needs to be declared before it gets used")
		fmt.Println(s.Error(expr.Token, errMsg))
		os.Exit(1)
	}

	return ident
}

func (s *TypeChecker) visitStructInstanceExpression(expr *parser.StructInstanceExpression) {

	identifier, ok := expr.Left.(*parser.Identifier)

	if ok {
		structDef, isMatched := s.Symbols.Resolve(identifier.Value)

		if !isMatched {
			errMsg := fmt.Sprintf("ERROR: struct (%v) needs to be defined, before creating instances of it", identifier.Value)
			s.Collector.Add(s.Error(identifier.Token, errMsg))
			return
		}

		body := expr.Body
		keys := structDef.DeclNode.(*parser.StructStatement)

		// means that some fields are left out of the having an associated value
		for _, field := range keys.Body {
			// find the key in the struct instance
			idx := slices.IndexFunc(body, func(f parser.FieldInstance) bool {
				return f.Key.Value == field.Key.Value
			})

			if idx == -1 {
				errMsg := fmt.Sprintf("ERROR: field (%v) needs to be instantiated with a value, cause it exists on the struct definition", field.Key.Value)
				s.Collector.Add(s.Error(field.Key.Token, errMsg))
			}
		}

		for _, field := range body {
			// find the key in the struct instance
			idx := slices.IndexFunc(keys.Body, func(f parser.Field) bool {
				return f.Key.Value == field.Key.Value
			})

			if idx == -1 {
				errMsg := fmt.Sprintf("ERROR: field (%v) doesn't exist on the struct definition, either add it to the definition or remove the field from the instance", field.Key.Value)
				s.Collector.Add(s.Error(field.Key.Token, errMsg))
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

	// nodeType := s.Symbols.CurrNode.Type.(*parser.MapType)
	// if nodeType.Type != "map" {
	// 	errMsg := fmt.Sprintf("ERROR: type mismatch in (%v) definition, defined as (%v), associated value (%v)", s.Symbols.CurrNode.Name, s.Symbols.CurrNode.Type, "map")
	// 	expr.Token.Text = expr.String()
	// 	s.Collector.Add(s.Error(expr.Token, errMsg))
	// 	return
	// }

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
		s.Collector.Add(s.Error(tok, errMsg))
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
	case *parser.BooleanLiteral:
		errMsg = "ERROR: can't use a boolean as an index, index can only be an int"
		tok = rf.Token
		tok.Text = rf.String()
	case *parser.StringLiteral:
		errMsg = "ERROR: can't use a string as an index, index can only be an int"
		tok = rf.Token
		tok.Text = rf.String()
	case *parser.FloatLiteral:
		errMsg = "ERROR: can't use a float as an index, index can only be an int"
		tok = rf.Token
		tok.Text = rf.String()
	default:

	}

	if len(tok.Text) > 0 {
		s.Collector.Add(s.Error(tok, errMsg))
	}
}

func (s *TypeChecker) visitMemberShipAccess(expr *parser.MemberShipExpression) {
	s.symbolReaderExpression(expr.Object)
	s.symbolReaderExpression(expr.Property)
}

// TODO: make an infer function to get the type of the associated value
// Then compare those types in the end result to see if they match or not

// parser expression here is one of 2 types, either parser.NodeType, or parser.MpaType
func (s *TypeChecker) inferAssociatedValueType(expr parser.Expression) parser.Type {

	switch ep := expr.(type) {
	// atomic types
	case *parser.StringLiteral:
		return &parser.NodeType{
			Type: parser.StringType,
		}
	case *parser.BooleanLiteral:
		return &parser.NodeType{
			Type: parser.BoolType,
		}
	case *parser.IntegerLiteral:
		return &parser.NodeType{
			Type: parser.IntType,
		}
	case *parser.FloatLiteral:
		return &parser.NodeType{
			Type: parser.FloatType,
		}

	case *parser.ArrayLiteral:
		return s.inferArrayType(ep).(*parser.NodeType)
	case *parser.MapLiteral:
		return s.inferMapType(ep)
	case *parser.Identifier:
		return s.inferIdentifierType(ep)
	case *parser.StructInstanceExpression:
		return s.inferStructInstanceType(ep)
	case *parser.IndexExpression:
		return s.inferIndexAccessType(ep).(*parser.NodeType)
	case *parser.CallExpression:
		return s.inferCallExpressionType(ep)
	case *parser.UnaryExpression:
		return s.inferUnaryExpressionType(ep)
	case *parser.BinaryExpression:
		return s.inferBinaryExpressionType(ep)
	}

	return &parser.NodeType{}
}

func (s *TypeChecker) inferArrayType(expr *parser.ArrayLiteral) parser.Type {

	if len(expr.Elements) == 0 {
		return s.Symbols.CurrNode.Type.(*parser.NodeType)
	}

	firstElem := &parser.NodeType{}

	for idx, elem := range expr.Elements {
		resType := s.inferAssociatedValueType(elem)
		if idx == 0 {
			fmt.Println(resType)
			firstElem = resType.(*parser.NodeType)
		}
		if firstElem.Type != resType.(*parser.NodeType).Type {
			errMsg := fmt.Sprintf("ERROR: multitude of different types in the array (%v,%v,...etc), remove incompatible types", firstElem, resType)
			expr.Token.Text = expr.String()
			s.insertUniqueErrors(s.Error(expr.Token, errMsg))
		}
	}

	return &parser.NodeType{
		Token:     firstElem.Token,
		Type:      "array",
		ChildType: firstElem,
	}
}

func (s *TypeChecker) inferMapType(expr *parser.MapLiteral) parser.Type {
	if len(expr.Pairs) == 0 {
		return s.Symbols.CurrNode.Type.(*parser.NodeType)
	}
	// use interface for readability (preferred over any)
	var keyElem interface{}
	var valElem interface{}
	idx := 0
	for key, value := range expr.Pairs {
		// key part
		resType := s.inferAssociatedValueType(key)
		if idx == 0 {
			keyElem = resType
		}
		switch rst := keyElem.(type) {
		case *parser.NodeType:
			if rst.Type != resType.GetType() {
				errMsg := fmt.Sprintf("ERROR: multitude of different types in the array (%v,%v,...etc), remove incompatible types", keyElem, resType)
				s.insertUniqueErrors(s.Error(key.GetToken(), errMsg))
			}
		case *parser.MapType:
			if rst.Type != resType.GetType() {
				errMsg := fmt.Sprintf("ERROR: multitude of different types in the array (%v,%v,...etc), remove incompatible types", keyElem, resType)
				s.insertUniqueErrors(s.Error(key.GetToken(), errMsg))
			}
		default:
		}

		// value part
		resType = s.inferAssociatedValueType(value)
		if idx == 0 {
			valElem = resType
			idx++
		}
		switch rst := valElem.(type) {
		case *parser.NodeType:
			if rst.Type != resType.GetType() {
				errMsg := fmt.Sprintf("ERROR: multitude of different types in the array (%v,%v,...etc), remove incompatible types", keyElem, resType)
				s.insertUniqueErrors(s.Error(value.GetToken(), errMsg))
			}
		case *parser.MapType:
			if rst.Type != resType.GetType() {
				errMsg := fmt.Sprintf("ERROR: multitude of different types in the array (%v,%v,...etc), remove incompatible types", keyElem, resType)
				s.insertUniqueErrors(s.Error(value.GetToken(), errMsg))
			}
		default:
		}
	}

	return &parser.MapType{
		Token: expr.Token,
		Type:  "map",
		Left:  keyElem.(parser.Type),
		Right: valElem.(parser.Type),
	}
}

func (s *TypeChecker) inferIdentifierType(expr *parser.Identifier) parser.Type {
	// call the visitIdentifier
	sym := s.visitIdentifier(expr)

	if sym == nil {
		return nil
	}

	switch node := sym.DeclNode.(type) {
	case *parser.LetStatement:
		return s.inferAssociatedValueType(node.Value)
	case *parser.StructInstanceExpression:
		return s.inferAssociatedValueType(node.Left)
	default:

	}
	return &parser.NodeType{
		Type: expr.Value,
	}
}

func (s *TypeChecker) inferStructInstanceType(expr *parser.StructInstanceExpression) parser.Type {
	// check if the types are compatible with the definition
	sym := s.visitIdentifier(expr.Left.(*parser.Identifier))

	if sym == nil {
		return nil
	}

	var keyElem interface{}
	idx := 0
	for id, elem := range expr.Body {
		// check the types are evaluated correctly on the fields
		resType := sym.DeclNode.(*parser.StructStatement).Body[id]
		if idx == 0 {
			keyElem = resType
			idx++
		}
		switch rst := keyElem.(type) {
		case *parser.NodeType:
			if rst.Type != resType.Value.String() {
				errMsg := fmt.Sprintf("ERROR: multitude of different types in the array (%v,%v,...etc), remove incompatible types", keyElem, resType)
				s.insertUniqueErrors(s.Error(elem.Value.GetToken(), errMsg))
			}
		case *parser.MapType:
			if rst.Type != resType.Value.String() {
				errMsg := fmt.Sprintf("ERROR: multitude of different types in the array (%v,%v,...etc), remove incompatible types", keyElem, resType)
				s.insertUniqueErrors(s.Error(elem.Value.GetToken(), errMsg))
			}
		default:
		}
	}

	return &parser.NodeType{
		Type: sym.Name,
	}
}

func (s *TypeChecker) inferIndexAccessType(expr *parser.IndexExpression) parser.Expression {
	// check what the left side is

	indexType := s.inferAssociatedValueType(expr.Index)

	if indexType.String() != "int" {
		errMsg := fmt.Sprintf("ERROR: can't use %v as index, index should be of type int", indexType)
		expr.Token.Text = expr.String()
		fmt.Println(s.Error(expr.Token, errMsg))
		return nil
	}

	return s.inferAssociatedValueType(expr.Left).(*parser.NodeType).ChildType
}

func (s *TypeChecker) inferCallExpressionType(expr *parser.CallExpression) parser.Type {
	sym := s.visitIdentifier(&expr.Function)

	if sym == nil {
		return nil
	}

	return sym.Type.(*parser.NodeType)
}

func (s *TypeChecker) inferUnaryExpressionType(expr *parser.UnaryExpression) parser.Type {
	if s.Symbols.CurrNode.Type.(*parser.NodeType).Type == parser.BoolType && expr.Operator == "-" {
		errMsg := "ERROR: can't use operator (-) with boolean types, only operator (!) is allowed"
		s.insertUniqueErrors(s.Error(expr.Token, errMsg))
	}

	return s.inferAssociatedValueType(expr.Right)
}

func (s *TypeChecker) inferBinaryExpressionType(expr *parser.BinaryExpression) parser.Type {
	// check if the operation is allowed on that type
	// rule: equality on all
	// comparison only on floats, and ints
	// rule: allow only comparison of the same types
	leftType := s.inferAssociatedValueType(expr.Left)
	rightType := s.inferAssociatedValueType(expr.Right)
	if leftType.String() != rightType.String() {
		errMsg := fmt.Sprintf(
			"ERROR: type mismatch, we can't compare 2 different types in a binary expression, left (%v), right (%v)", leftType, rightType,
		)
		expr.Token.Col++
		fmt.Println(s.Error(expr.Token, errMsg))
		os.Exit(1)
	}

	operator := expr.Operator

	switch operator {
	case "==", "!=":
		return &parser.NodeType{
			Token: expr.Token,
			Type:  parser.BoolType,
		}
	case ">=", "<=", ">", "<":
		fmt.Println(leftType)
		switch leftType.String() {
		case parser.StringType:
			// error
			errMsg := fmt.Sprintf(
				"ERROR: (%s) isn't allowed on string type", operator,
			)
			s.insertUniqueErrors(s.Error(expr.Token, errMsg))
		case parser.BoolType:
			errMsg := fmt.Sprintf(
				"ERROR: (%s) isn't allowed on boolean type", operator,
			)
			s.insertUniqueErrors(s.Error(expr.Token, errMsg))
			// error
		default:
			return &parser.NodeType{
				Token: expr.Token,
				Type:  parser.BoolType,
			}
		}
	case "+":
		if leftType.String() == parser.BoolType {
			errMsg := fmt.Sprintf(
				"ERROR: (%s) isn't allowed on boolean type", operator,
			)
			s.insertUniqueErrors(s.Error(expr.Token, errMsg))
		} else {
			return &parser.NodeType{
				Token: expr.Token,
				Type:  leftType.String(),
			}
		}
	case "-", "/", "*", "%":
		switch leftType.String() {
		case parser.StringType:
			// error
			errMsg := fmt.Sprintf(
				"ERROR: (%s) isn't allowed on string type", operator,
			)
			s.insertUniqueErrors(s.Error(expr.Token, errMsg))
		case parser.BoolType:
			errMsg := fmt.Sprintf(
				"ERROR: (%s) isn't allowed on boolean type", operator,
			)
			s.insertUniqueErrors(s.Error(expr.Token, errMsg))
			// error
		}
	case "&&", "||":
		if leftType.String() != parser.BoolType {
			errMsg := fmt.Sprintf(
				"ERROR: (%s) isn't allowed on %s type", operator, leftType.String(),
			)
			s.insertUniqueErrors(s.Error(expr.Token, errMsg))
		}
	default:

	}

	return &parser.NodeType{
		Token: expr.Token,
		Type:  parser.BoolType,
	}
}
