package semantics

import (
	"blk/parser"
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
	DeclNode  parser.Node // pointer to node dcl in AST
	Kind      SymbolKind  // func, var, param, let...
	Type      parser.Expression
	IsMutable bool
	Depth     int
}

type symbolTable struct {
	Parent *symbolTable          // for nested scopes
	Store  map[string]SymbolInfo // current scope's entries
	Depth  int
}

func NewSymbolTable() *symbolTable {
	return &symbolTable{
		Store: make(map[string]SymbolInfo),
	}
}

type symbolResolver struct {
	current *symbolTable
}

func NewSymbolResolver() *symbolResolver {
	return &symbolResolver{
		current: NewSymbolTable(),
	}
}

func (s *symbolResolver) Define(name string, sym *SymbolInfo) {
	if _, ok := s.current.Store[name]; ok {
		// if already exist we skip
		return
	}
	s.current.Store[name] = *sym
}

func (s *symbolResolver) Resolve(name string) (*SymbolInfo, bool) {
	scope := s.current
	for scope != nil {
		if sym, ok := scope.Store[name]; ok {
			return &sym, true
		}
		scope = scope.Parent
	}
	return nil, false
}

func (s *symbolResolver) EnterScope() *symbolTable {
	newScope := NewSymbolTable()
	newScope.Parent = s.current
	newScope.Depth = s.current.Depth + 1
	s.current = newScope
	return newScope
}

func (s *symbolResolver) ExitScope(curr *symbolTable) *symbolTable {
	if curr.Parent != nil {
		s.current = curr.Parent
	}
	return s.current
}
