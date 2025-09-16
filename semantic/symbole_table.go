package semantic

import (
	"blk/ast"
	"fmt"
)

type Symbol struct {
	Name       string
	Kind       ast.Type
	IsMutable  bool
	IsFunction bool
	DeclNode   ast.Node
}

type Scope struct {
	Parent  *Scope
	Symbols map[string]*Symbol
}

type SymbolTable struct {
	CurrentScope *Scope
	GlobalScope  *Scope
}

func NewSymTable() *SymbolTable {
	scope := &Scope{
		Symbols: make(map[string]*Symbol),
	}
	return &SymbolTable{
		CurrentScope: scope,
		GlobalScope:  scope,
	}
}

// enter a new scope
func (st *SymbolTable) EnterScope() {
	st.CurrentScope = &Scope{
		Parent:  st.CurrentScope,
		Symbols: map[string]*Symbol{},
	}
}

// exit the current scope to the parent scope (prev one)
func (st *SymbolTable) ExitScope() {
	if st.CurrentScope.Parent != nil {
		st.CurrentScope = st.CurrentScope.Parent
	}
}

// define a new symbol in the symbol store
func (s *Scope) Define(name string, sym *Symbol) error {
	if _, ok := s.Symbols[name]; ok {
		return fmt.Errorf("%s already exists in the scope, u can't redeclare it", name)
	}

	s.Symbols[name] = sym
	return nil
}

func (s *Scope) Update(name string, sym *Symbol) error {
	_, ok := s.Symbols[name]
	if !ok {
		return fmt.Errorf("symbol not found in the current scope")
	}

	s.Symbols[name] = sym
	return nil
}

// recursive search on the symbol name
func (s *Scope) Resolve(name string) *Symbol {
	if sym, ok := s.Symbols[name]; ok {
		return sym
	}
	if s.Parent != nil {
		return s.Parent.Resolve(name)
	}
	return nil // not found
}
