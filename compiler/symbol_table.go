package compiler

import (
	"blk/parser"
	"fmt"
)

type SymbolKind = string

const (
	SymbolVar    SymbolKind = "var"
	SymbolLet    SymbolKind = "let"
	SymbolFunc   SymbolKind = "fn"
	SymbolStruct SymbolKind = "struct"
	SymbolType   SymbolKind = "type"
)

type Position struct {
	Row int
	Col int
}

type Type interface{}

type SymbolInfo struct {
	Type
	Name      string
	DeclPos   Position   // where declared
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
	switch node.(type) {
	case *parser.LetStatement:
		s.visitVarDCL(node.(*parser.LetStatement))
	case *parser.FunctionStatement:
		s.visitFuncDCL(node.(*parser.FunctionStatement))
	default:
	}
}

func (s *SymbolTable) visitFuncDCL(node *parser.FunctionStatement) {
	sym := &SymbolInfo{
		Name:  node.Name,
		Kind:  SymbolFunc,
		Depth: s.DepthIndicator,
		Type:  nil, // Placeholder
	}
	_, ok := s.Resolve(sym.Name)
	if ok {
		errMsg := fmt.Errorf("ERROR: %v identifer is already declared", sym.Name)
		fmt.Println(errMsg)
		return
	}

	if node.Body != nil {
		nwTab := NewSymbolTable()
		nwTab.Parent = s
		nwTab.DepthIndicator = s.DepthIndicator + 1
		for _, nd := range node.Body.Body {
			nwTab.symbolReader(nd)
		}
	}

	s.Define(sym.Name, sym)
}

func (s *SymbolTable) visitVarDCL(node *parser.LetStatement) {
	kind := SymbolLet

	if node.Token.Text == "var" {
		kind = SymbolVar
	}

	sym := &SymbolInfo{
		Name:  node.Name.Value,
		Kind:  kind,
		Depth: s.DepthIndicator,
		Type:  nil, // Placeholder
	}

	_, ok := s.Resolve(sym.Name)

	if ok {
		errMsg := fmt.Errorf("ERROR: %v identifer is already declared", sym.Name)
		fmt.Println(errMsg)
		return
	} else {
		s.Define(sym.Name, sym)
	}
}
