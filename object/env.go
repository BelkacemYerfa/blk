package object

type Environment struct {
	outer *Environment
	store map[string]Object
}

func NewEnvironment(outer *Environment) *Environment {
	s := make(map[string]Object)
	return &Environment{
		outer: outer,
		store: s,
	}
}

func (e *Environment) Resolve(name string) (Object, bool) {
	obj, ok := e.store[name]
	if !ok && e.outer != nil {
		obj, ok = e.outer.Resolve(name)
	}
	return obj, ok
}

func (e *Environment) Define(name string, val Object) Object {
	if _, ok := e.store[name]; ok {
		return val
	}
	// define if there no value already bound to it
	e.store[name] = val
	return val
}
