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
	if _, exists := s.current.Store[name]; exists {
		// TODO: add proper error reporting in here
		return
	}

	s.current.Store[name] = *sym
}

func (s *symbolResolver) Resolve(name string) (*SymbolInfo, bool) {
	if sym, ok := s.current.Store[name]; ok {
		// if sym.Depth == s.current.Depth {
		return &sym, true
		// }
	}
	if s.current.Parent != nil {
		s.current = s.current.Parent
		sym, _ := s.Resolve(name)
		if sym != nil {
			return sym, true
		}
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

func (s *symbolResolver) ExitScope(parent *symbolTable) *symbolTable {
	s.current = parent
	return s.current
}
