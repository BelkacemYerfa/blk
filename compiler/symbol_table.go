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

type SymbolScope string

const (
	GlobalScope SymbolScope = "Global"
	LocalScope  SymbolScope = "Local"
)

type Position struct {
	Row int
	Col int
}

type Type interface {}

type SymbolInfo struct {
	Type
	Name      string
	DeclPos   Position   // where declared
	Kind      SymbolKind // func, var, param, let...
	IsMutable bool
	Scope     SymbolScope
	Index     int
}

type SymbolTable struct {
	Parent         *SymbolTable          // for nested scopes
	Store          map[string]SymbolInfo // current scope's entries
	NumDefinitions int
	Errors []error
}

func NewSymbolTable() *SymbolTable {
	return &SymbolTable{
		Store: make(map[string]SymbolInfo),
	}
}


func (s *SymbolTable) Define(name string) SymbolInfo {
	symbol := SymbolInfo{Name: name, Index: s.NumDefinitions, Scope: GlobalScope}
	s.Store[name] = symbol
	s.NumDefinitions++
	return symbol
}

func (s *SymbolTable) Lookup(name string) (*SymbolInfo, bool) {
	curr := s
	for curr != nil {
		if sym, ok := curr.Store[name]; ok {
			return &sym, true
		}
		curr = curr.Parent
	}
	return nil, false
}

func (s *SymbolTable) Resolve(name string) (SymbolInfo, bool) {
	result, ok := s.Store[name]
	return result, ok
}

func (s *SymbolTable) SymboleBuilder(ast *parser.Program) {
	for _, node := range ast.Statements {

		switch node.(type) {
		case *parser.LetStatement:
			s.visitVarDCL(node.(*parser.LetStatement))
		default:
		}

	}
}


func (s *SymbolTable) visitVarDCL(node *parser.LetStatement) {
	sym := &SymbolInfo{
		Name: node.Name.Value,
		Kind: SymbolLet,
		Type: nil, // Placeholder
	}
	_, ok := s.Resolve(sym.Name)
	if ok {
		errMsg := fmt.Errorf("ERROR: %v identifer is already declared", sym.Name)
		s.Errors = append(s.Errors, errMsg)
	} else {
		s.Define(sym.Name)
	}
}
