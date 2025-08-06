package object

type ItemObject struct {
	Object
	IsMutable bool
	IsBuiltIn bool
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

func (e *Environment) Resolve(name string) (ItemObject, bool) {
	obj, ok := e.store[name]
	if !ok && e.outer != nil {
		obj, ok = e.outer.Resolve(name)
	}
	return obj, ok
}

func (e *Environment) Define(name string, val ItemObject) Object {
	if _, ok := e.store[name]; ok {
		return val
	}
	// define if there no value already bound to it
	e.store[name] = val
	return val
}
