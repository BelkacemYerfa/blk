package object

type ItemObject struct {
	Object
	IsMutable bool
	IsBuiltIn bool // this is useful for builtin function & default value into the language it self
}

type Environment struct {
	outer *Environment
	store map[string]ItemObject
}

func NewEnvironment(outer *Environment) *Environment {
	s := make(map[string]ItemObject)
	return &Environment{
		outer: outer,
		store: s,
	}
}

func (e *Environment) GetStore() map[string]ItemObject {
	return e.store
}

func (e *Environment) Resolve(name string) (ItemObject, bool) {
	obj, ok := e.store[name]
	if !ok && e.outer != nil {
		obj, ok = e.outer.Resolve(name)
	}
	return obj, ok
}

func (e *Environment) Define(name string, val ItemObject) (Object, bool) {
	if _, ok := e.store[name]; ok {
		return val, true
	}
	// define if there no value already bound to it
	e.store[name] = val
	// second return types is to indicate if the value is already there or first declare
	return val, false
}

func (e *Environment) OverrideDefine(name string, val ItemObject) Object {
	e.store[name] = val
	return val
}

func (e *Environment) GetOuterScope() *Environment {
	return e.outer
}
