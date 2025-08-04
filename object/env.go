package object

type Environment struct {
	parent *Environment
	store  map[string]Object
}

func NewEnvironment(parent *Environment) *Environment {
	s := make(map[string]Object)
	return &Environment{
		parent: parent,
		store:  s,
	}
}

func (e *Environment) Resolve(name string) (Object, bool) {
	obj, ok := e.store[name]
	if !ok && e.parent != nil {
		obj, ok = e.parent.Resolve(name)
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

func (e *Environment) GetParentEnv() *Environment {
	return e.parent
}
