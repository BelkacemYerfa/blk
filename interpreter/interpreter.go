package interpreter

import (
	"blk/ast"
	"blk/lexer"
	"blk/object"
	"blk/stdlib"
	"fmt"
	"strings"
)

var (
	TRUE  = &object.Boolean{Value: true}
	FALSE = &object.Boolean{Value: false}
)

type Interpreter struct {
	env *object.Environment
}

func NewInterpreter(env *object.Environment) *Interpreter {
	if env == nil {
		env = object.NewEnvironment(nil)
	}
	return &Interpreter{
		env: env,
	}
}

func (i *Interpreter) enterScope() {
	newScope := object.NewEnvironment(i.env)
	i.env = newScope
}

func (i *Interpreter) exitScope() {
	i.env = i.env.GetOuterScope()
}

type LEVEL = int

const (
	WARNING LEVEL = iota
	ERROR
)

func newError(level LEVEL, format string, a ...interface{}) *object.Error {
	prefix := "ERROR"

	if level == WARNING {
		prefix = "WARNING"
	}

	msg := fmt.Sprintf(`%s: %s`, prefix, format)
	return &object.Error{Message: fmt.Sprintf(msg, a...)}
}

func isError(obj object.Object) bool {
	if obj != nil {
		return obj.Type() == object.ERROR_OBJ
	}
	return false
}

func (i *Interpreter) Eval(node ast.Node) object.Object {
	switch nd := node.(type) {
	case *ast.Program:
		return i.evalProgram(nd.Statements)

	case *ast.ImportStatement:
		module, ok := stdlib.BuiltinModules[nd.ModuleName.Value]
		if !ok {
			return newError(ERROR, "Module Not found %s", nd.ModuleName)
		}
		attrs := make(map[string]object.Object, len(module))

		for name, fn := range module {
			// means that this function is internal and can't be used
			// doesn't make so much sense cause u can just not register them
			if strings.HasPrefix(name, "_") {
				continue
			}
			// register function so u it can be used
			attrs[name] = object.ItemObject{
				Object:    fn,
				IsBuiltIn: true,
			}
		}

		_, firstDeclaration := i.env.Define(nd.ModuleName.Value, object.ItemObject{
			Object: &object.BuiltInModule{
				Name:  nd.ModuleName.Value,
				Attrs: attrs,
			},
			IsBuiltIn: true,
		})

		if firstDeclaration {
			return newError(WARNING, "found module name %s, is used as a declaration, consider renaming it in the declaration to something else", nd.ModuleName)
		}

	case *ast.StructExpression:
		methods := make(map[string]object.Object, 0)
		fields := make(map[string]object.Object, 0)

		// var declaration such as (x := 0, name :: "john")
		for _, decl := range nd.Fields {
			val := i.Eval(decl.Value)
			if isError(val) {
				return val
			}
			varDecl := object.ItemObject{}
			switch v := val.(type) {
			// this is in case for the val is another var declaration
			// this ensures to make a copy of the value and not the
			case object.ItemObject:
				valueClone := object.DeepCopy(v.Object)
				varDecl = object.ItemObject{
					Object: valueClone,
				}
				if decl.Token.Text == "let" {
					varDecl.IsMutable = true
				}
			default:
				// define it in the scope
				varDecl = object.ItemObject{
					Object: v,
				}
				if decl.Token.Text == "let" {
					varDecl.IsMutable = true
				}
			}

			fields[decl.Name.Value] = varDecl
		}

		// methods built in into the struct
		for _, method := range nd.Methods {
			// here we pass teh value cause it is of type ast.FunctionExpression
			evaluated := i.Eval(method.Value)
			methods[method.Key.Value] = object.ItemObject{
				Object: evaluated,
			}
		}

		return &object.Struct{
			Fields:  fields,
			Methods: methods,
		}

	case *ast.StructInstanceExpression:
		// deal with this one
		// get the current left side, since it is an identifier
		val := i.Eval(nd.Left)
		if isError(val) {
			return val
		}
		// val mostly is struct name
		// checks the fields also compare
		// for now the fields are mutable, no support for const :: in fields
		castDef, _ := object.Cast(val)
		structDef := castDef.(*object.Struct)
		copyOfStructDef := object.DeepCopy(structDef).(*object.Struct)

		// only fields which are allowed to get mutated
		// methods are not allowed
		for _, field := range nd.Body {
			_, ok := structDef.Methods[field.Key.Value]
			if ok {
				return newError(ERROR, "methods of a struct can't be mutated")
			}

			fieldDef, ok := structDef.Fields[field.Key.Value]
			if !ok {
				return newError(ERROR, "%s field doesn't exist on the struct definition, consider declaring it", field.Key.Value)
			}

			fieldValue := i.Eval(field.Value)

			if fieldDef.Type() != fieldValue.Type() {
				// type error
				return newError(ERROR, "type mismatch on %s, definition type %s, got %s", field.Key.Value, fieldDef.Type(), fieldValue.Type())
			}

			varDecl := object.ItemObject{
				IsMutable: true,
			}
			switch v := fieldValue.(type) {
			// this is in case for the val is another var declaration
			// this ensures to make a copy of the value and not use a reference to the value
			case object.ItemObject:
				varDecl.Object = object.DeepCopy(v.Object)
			default:
				// define it in the scope
				varDecl.Object = v
			}

			copyOfStructDef.Fields[field.Key.Value] = varDecl

		}

		return copyOfStructDef

	case *ast.ExpressionStatement:
		return i.Eval(nd.Expression)

	case *ast.IntegerLiteral:
		return &object.Integer{
			Value: nd.Value,
		}
	case *ast.FloatLiteral:
		return &object.Float{
			Value: nd.Value,
		}
	case *ast.StringLiteral:
		return &object.String{
			Value: nd.Value,
		}
	case *ast.BooleanLiteral:
		return nativeBooleanObject(nd.Value)

	case *ast.ArrayLiteral:
		elements := i.evalArrayExpression(nd.Elements)
		if len(elements) == 1 && isError(elements[0]) {
			return elements[0]
		}
		return &object.Array{Elements: elements}

	case *ast.MapLiteral:
		return i.evalMapExpression(nd.Pairs)

	case *ast.IndexExpression:
		left := i.Eval(nd.Left)
		if isError(left) {
			return left
		}
		index := i.Eval(nd.Index)
		if isError(index) {
			return index
		}
		return i.evalIndexExpression(left, index)

	case *ast.WhileStatement:
		return i.evalWhileStatement(nd)

	case *ast.ForStatement:
		return i.evalForStatement(nd)

	case *ast.SkipStatement:
		return &object.Skip{}

	case *ast.VarDeclaration:
		val := i.Eval(nd.Value)
		if isError(val) {
			return val
		}

		castedVal, _ := object.Cast(val)

		// this tells the interpreter that those type of values aren't allowed to be const
		if (castedVal.Type() == object.ARRAY_OBJ || castedVal.Type() == object.MAP_OBJ) && nd.Token.Text == "const" {
			return newError(ERROR, "%v isn't allowed to be an const, consts are only: ints, floats, strings, booleans", castedVal.Type())
		}

		newVal := object.ItemObject{
			Object: castedVal,
		}

		if nd.Token.Text == "let" {
			newVal.IsMutable = true
		}

		switch v := castedVal.(type) {
		// means that this types are give u a deep copy of their value
		case *object.Float, *object.Integer, *object.String, *object.Boolean:
			newVal.Object = object.DeepCopy(v)

		// means that this types are being shallow copied
		case *object.Array, *object.Map, *object.Struct:
			newVal.Object = v
		}

		// define it in the scope
		_, firstDeclaration := i.env.Define(nd.Name.Value, newVal)

		if firstDeclaration {
			return newError(WARNING, "name %s is already in use", nd.Name.Value)
		}

	case *ast.Identifier:
		return i.evalIdentifier(nd)

	case *ast.FunctionExpression:
		params := nd.Args
		body := nd.Body
		if len(nd.Self.Value) > 0 {
			params = append([]*ast.Identifier{nd.Self}, params...)
		}
		return &object.Function{Parameters: params, Env: i.env, Body: body}

	case *ast.CallExpression:
		function := i.Eval(&nd.Function)
		if isError(function) {
			return function
		}
		// function is always of type object.ItemObject
		ableToCast := function.(object.ItemObject).IsBuiltIn
		args := i.evalExpressions(nd.Args, !ableToCast)
		if len(args) == 1 && isError(args[0]) {
			// error out
			return args[0]
		}
		return i.applyFunction(function, args)

	case *ast.ReturnStatement:
		val := i.Eval(nd.ReturnValue)
		if isError(val) {
			return val
		}
		return &object.ReturnValue{Value: val}

	case *ast.ScopeStatement:
		var evaluation object.Object
		i.enterScope()
		{
			// scope makes it easier to understand
			evaluation = i.evalBlockStatement(nd.Body)
		}
		i.exitScope()
		return evaluation

	case *ast.BlockStatement:
		var evaluation object.Object
		i.enterScope()
		{
			// scope makes it easier to understand
			evaluation = i.evalBlockStatement(nd)
		}
		i.exitScope()
		return evaluation

	case *ast.IfExpression:
		return i.evalIfExpression(nd)

	case *ast.UnaryExpression:
		right := i.Eval(nd.Right)
		if isError(right) {
			return right
		}
		return i.evalUnaryExpression(nd.Operator, right)

	case *ast.BinaryExpression:
		left := i.Eval(nd.Left)
		if isError(left) {
			return left
		}
		right := i.Eval(nd.Right)
		if isError(right) {
			return right
		}

		return i.evalBinaryExpression(nd.Operator, nd.Left, left, right)

	case *ast.MemberShipExpression:
		// evaluate the owner
		obj := i.Eval(nd.Object)
		if isError(obj) {
			fmt.Println(obj)
			return obj
		}

		return i.evalMembershipExpression(obj, nd.Object, nd.Property)
	default:
	}
	return nil
}

func (i *Interpreter) evalProgram(stmts []ast.Statement) object.Object {
	var result object.Object
	for _, statement := range stmts {
		result = i.Eval(statement)
		res, _ := object.Cast(result)
		switch res := res.(type) {
		case *object.ReturnValue:
			return res.Value
		case *object.Error:
			return result
		}
	}

	return result
}

func nativeBooleanObject(val bool) *object.Boolean {
	if val {
		return TRUE
	}
	return FALSE
}

func (i *Interpreter) evalExpressions(exps []ast.Expression, ableToCast bool) []object.Object {
	var result []object.Object
	for _, e := range exps {
		evaluated := i.Eval(e)
		if isError(evaluated) {
			return []object.Object{evaluated}
		}
		argEval := evaluated
		switch ev := evaluated.(type) {
		case object.ItemObject:
			// leave it as it is cause it is in the proper shape
			argEval = object.ItemObject{
				Object: ev.Object,
				// means any param that gets passed to the func is mutable
				// predefined values that u pass are not affected by this
				IsMutable: true,
			}
		default:
			// wrap it inside of ItemObject struct
			argEval = object.ItemObject{
				Object: object.DeepCopy(ev),
				// means any param that gets passed to the func is mutable
				// predefined values that u pass are not affected by this
				IsMutable: true,
			}
		}
		if ableToCast {
			argEval, _ = object.Cast(evaluated)
		}
		result = append(result, argEval)
	}
	return result
}

// this is used for evaluating array elements
func (i *Interpreter) evalArrayExpression(exps []ast.Expression) []object.Object {
	result := make([]object.Object, 0, len(exps))
	var firstElem object.Object
	for idx, e := range exps {
		evaluated := i.Eval(e)
		if isError(evaluated) {
			return []object.Object{evaluated}
		}
		elemEval, _ := object.Cast(evaluated)
		if idx == 0 {
			firstElem = elemEval
		}
		if firstElem.Type() != elemEval.Type() {
			// throw an error here
			return []object.Object{
				newError(ERROR, "multitude of types, (%v,%v), array elements should be of one type", firstElem.Type(), elemEval.Type()),
			}
		}
		// else push the element
		result = append(result, evaluated)
	}
	return result
}

// this is used for evaluating map pairs (key, value)
func (i *Interpreter) evalMapExpression(prs map[ast.Expression]ast.Expression) object.Object {
	pairs := make(map[object.HashKey]object.HashPair, len(prs))
	var keyEl, valEl object.Object
	idx := 0
	for keyNode, valueNode := range prs {
		key := i.Eval(keyNode)
		if isError(key) {
			return key
		}
		key, _ = object.Cast(key)
		hashKey, ok := key.(object.Hashable)
		if !ok {
			return newError(ERROR, "unusable as hash key: %s", key.Type())
		}
		if idx == 0 {
			keyEl = key
		}
		if keyEl.Type() != key.Type() {
			return newError(ERROR, "multitude of types, (%v,%v), key elements of a map should be of one type", keyEl.Type(), key.Type())
		}

		value := i.Eval(valueNode)
		if isError(value) {
			return value
		}
		hashed := hashKey.HashKey()
		value, _ = object.Cast(value)
		if idx == 0 {
			valEl = value
		}
		if valEl.Type() != value.Type() {
			return newError(ERROR, "multitude of types, (%v,%v), value elements of a map should be of one type", valEl.Type(), value.Type())
		}
		pairs[hashed] = object.HashPair{Key: key, Value: value}
		idx++
	}

	return &object.Map{Pairs: pairs}
}

func (i *Interpreter) evalIndexExpression(lf, index object.Object) object.Object {
	// make sure that the left is either a map or an array
	lf, _ = object.Cast(lf)
	index, _ = object.Cast(index)
	switch lf := lf.(type) {
	case *object.Array:
		if index.Type() != object.INTEGER_OBJ {
			return newError(ERROR, "index side needs to be an integer, got %v", index.Type())
		}
		return i.evalArrayIndexExpression(lf, index)

	case *object.Map:
		return i.evalMapIndexExpression(lf, index)

	default:
		return newError(ERROR, "left side needs to be either an array or map, got %v", lf.Type())
	}
}

func (i *Interpreter) evalArrayIndexExpression(array, index object.Object) object.Object {
	arrayObject := array.(*object.Array)
	idx := index.(*object.Integer).Value
	max := int64(len(arrayObject.Elements) - 1)

	if idx < 0 || idx > max {
		return newError(ERROR, "index out of bound, %d", idx)
	}

	return arrayObject.Elements[idx]
}

func (i *Interpreter) evalMapIndexExpression(hashMap, index object.Object) object.Object {
	mapObject := hashMap.(*object.Map)

	key, ok := index.(object.Hashable)
	if !ok {
		return newError(ERROR, "unusable as hash key: %s", index.Type())
	}

	pair, ok := mapObject.Pairs[key.HashKey()]
	if !ok {
		return newError(ERROR, "index (%v) is not associated with any value", index.Inspect())
	}

	return pair.Value
}

func (i *Interpreter) evalIdentifier(identifier *ast.Identifier) object.Object {

	// do the check on the operation layer if the current treated value is mutable or not
	if obj, ok := i.env.Resolve(identifier.Value); ok {
		return obj
	}

	if buildInFunc, ok := builtInFunction[identifier.Value]; ok {
		return object.ItemObject{
			Object:    buildInFunc,
			IsBuiltIn: true,
		}
	}

	if builtInCons, ok := builtInConstants[identifier.Value]; ok {
		return object.ItemObject{
			Object:    builtInCons,
			IsBuiltIn: true,
		}
	}

	return newError(ERROR, "identifier not found: %s", identifier.Value)
}

func (i *Interpreter) applyFunction(fn object.Object, args []object.Object) object.Object {

	// function cast
	fn, _ = object.Cast(fn)
	switch fn := fn.(type) {
	case *object.Function:
		argSize := len(args)
		fnParamSize := len(fn.Parameters)

		// check that the number of params is the same
		if argSize != fnParamSize {
			return newError(ERROR, "wrong number of arguments. got=%d, want=%d",
				argSize, fnParamSize)
		}

		extendedEnv := extendFunctionEnv(fn, args)
		// save the current env
		previousEnv := i.env
		i.env = extendedEnv
		evaluated := i.Eval(fn.Body)
		// restore the old env
		i.env = previousEnv
		return unwrapReturnValue(evaluated)

	case *object.BuiltinFn:
		return fn.Fn(args...)

	default:
		return newError(ERROR, "not a function: %s", fn.Type())
	}

}

func extendFunctionEnv(
	fn *object.Function,
	args []object.Object,
) *object.Environment {
	env := object.NewEnvironment(fn.Env)

	for paramIdx, param := range fn.Parameters {
		if param.Value == lexer.TokenSelf {
			// 0 is the first context of the current struct
			env.Define(param.Value, object.ItemObject{
				Object: args[0],
			})
		} else {
			env.Define(param.Value, object.ItemObject{
				Object: args[paramIdx],
				// this makes the params mutable
				IsMutable: true,
			})
		}
	}

	return env
}

func unwrapReturnValue(obj object.Object) object.Object {
	if returnValue, ok := obj.(*object.ReturnValue); ok {
		return returnValue.Value
	}
	return obj
}

func (i *Interpreter) evalBlockStatement(block *ast.BlockStatement) object.Object {
	var result object.Object

	for _, statement := range block.Body {
		result = i.Eval(statement)
		if result != nil {
			rt := result.Type()
			if rt == object.RETURN_VALUE_OBJ || rt == object.ERROR_OBJ {
				return result
			}
		}
	}

	return result
}

func (i *Interpreter) evalForStatement(nd *ast.ForStatement) object.Object {
	// check that the target is either an array or a map
	target, _ := object.Cast(i.Eval(nd.Target))

	switch tg := target.(type) {
	case *object.Array:
		// max : 2 identifier
		// 1: indicates the index of the current value
		// 2: indicates the current value
		if len(tg.Elements) == 0 {
			// skip don't do anything
			return nil
		}

		i.enterScope()
		{
			// init the index with value 0
			// init the value with first elem value
			for idx, elem := range tg.Elements {
				// first identifier (index) - always present
				if len(nd.Identifiers) >= 1 && nd.Identifiers[0].Value != "_" {
					i.env.OverrideDefine(nd.Identifiers[0].Value, object.ItemObject{
						Object: &object.Integer{Value: int64(idx)},
					})
				}

				// second identifier (value) - optional
				if len(nd.Identifiers) >= 2 {
					newVal := object.DeepCopy(elem)
					i.env.OverrideDefine(nd.Identifiers[1].Value, object.ItemObject{
						Object: newVal,
					})
				}

				// evaluate the body
				res := i.evalBlockStatement(nd.Body)
				if res != nil {
					switch res.Type() {
					case object.RETURN_VALUE_OBJ:
						// early return
						return res
					case object.SKIP_OBJ:
						// skip to the next iteration
						continue
					case object.ERROR_OBJ:
						return res
					}
				}
			}
		}
		i.exitScope()

	case *object.Map:
		// max : 2 identifier
		// 1: indicates the key
		// 2: indicates the value associated with the key

		if len(tg.Pairs) == 0 {
			// skip
			return nil
		}

		i.enterScope()
		{
			for _, elem := range tg.Pairs {
				// first identifier (key) - always present
				if len(nd.Identifiers) >= 1 && nd.Identifiers[0].Value != "_" {
					newVal := object.DeepCopy(elem.Key)
					i.env.OverrideDefine(nd.Identifiers[0].Value, object.ItemObject{
						Object: newVal,
					})
				}

				// second identifier (value) - optional
				if len(nd.Identifiers) >= 2 {
					newVal := object.DeepCopy(elem.Value)
					i.env.OverrideDefine(nd.Identifiers[1].Value, object.ItemObject{
						Object: newVal,
					})
				}

				// evaluate the body
				res := i.evalBlockStatement(nd.Body)
				if res != nil {
					switch res.Type() {
					case object.RETURN_VALUE_OBJ:
						// early return
						return res
					case object.SKIP_OBJ:
						// skip to the next iteration
						continue
					case object.ERROR_OBJ:
						return res
					}
				}
			}
		}
		i.exitScope()

	default:
		return newError(ERROR, "target needs to be either an array or a map, got %v", tg.Type())
	}

	return nil
}

func (i *Interpreter) evalWhileStatement(nd *ast.WhileStatement) object.Object {
	condition := i.Eval(nd.Condition)

	if isError(condition) {
		return condition
	}

	condition, _ = object.Cast(condition)

	switch cdn := condition.(type) {
	case *object.Boolean:
		// continue until the condition is broken
		for cdn.Value {
			// evaluate the body
			res := i.Eval(nd.Body)
			if res != nil {
				switch res.Type() {
				case object.RETURN_VALUE_OBJ:
					// early return
					return res
				case object.SKIP_OBJ:
					// skip to the next iteration
					continue
				case object.ERROR_OBJ:
					return res
				}
			}
			cdn = i.Eval(nd.Condition).(*object.Boolean)

		}
	default:
		// error out
		return newError(ERROR, "evaluation of the condition in a while loop needs to return a boolean not %s", cdn)
	}

	// maybe
	return nil
}

func (i *Interpreter) evalIfExpression(nd *ast.IfExpression) object.Object {
	condition := i.Eval(nd.Condition)

	if isError(condition) {
		return condition
	}

	condition, _ = object.Cast(condition)

	switch cdn := condition.(type) {
	case *object.Boolean:
		// continue
		if cdn.Value {
			// eval the consequence
			return i.Eval(nd.Consequence)
		} else {
			// eval the alternative
			return i.Eval(nd.Alternative)
		}
	default:
		// error out

		return newError(ERROR, "evaluation of the condition needs to return a boolean not %s", cdn)
	}
}

func (i *Interpreter) evalUnaryExpression(op string, right object.Object) object.Object {
	switch op {
	case lexer.TokenExclamation:
		// check the right side
		if right.Type() == object.BOOLEAN_OBJ {
			return i.evalBangOperatorExpression(right)
		}
		// throw an error
		return newError(ERROR, "! operator can only be applied on boolean values")
	case lexer.TokenMinus:
		// support for both ints and floats
		return i.evalMinusPrefixOperatorExpression(right)
	default:
	}

	return newError(ERROR, "unknown operator: %s%s", op, right.Type())
}

func (i *Interpreter) evalBangOperatorExpression(right object.Object) *object.Boolean {
	rt, _ := object.Cast(right)
	r := rt.(*object.Boolean)

	if r.Value {
		return FALSE
	}

	return TRUE
}

func (i *Interpreter) evalMinusPrefixOperatorExpression(right object.Object) object.Object {
	rt, _ := object.Cast(right)

	switch right := rt.(type) {
	case *object.Integer:
		value := right.Value
		return &object.Integer{
			Value: -value,
		}
	case *object.Float:
		value := right.Value
		return &object.Float{
			Value: -value,
		}
	default:
		// throw an error
		return newError(ERROR, "unknown operator: -%s", right.Type())
	}
}

func (i *Interpreter) evalBinaryExpression(op string, leftNode ast.Expression, left, right object.Object) object.Object {

	switch {
	case left.Type() == object.INTEGER_OBJ && right.Type() == object.INTEGER_OBJ:
		return i.evalIntegerInfixExpression(op, left, right)

	case left.Type() == object.FLOAT_OBJ && right.Type() == object.FLOAT_OBJ:
		return i.evalFloatInfixExpression(op, left, right)

	case left.Type() == object.BOOLEAN_OBJ && right.Type() == object.BOOLEAN_OBJ:
		// this not allowed at all (no operations on booleans)
		// the only op allowed are (&&, ||)
		return i.evalBooleanInfixExpression(op, left, right)

	case left.Type() == object.STRING_OBJ || right.Type() == object.STRING_OBJ:
		// allow addition with anything
		return i.evalStringInfixExpression(op, left, right)

	case left.Type() == object.STRUCT_OBJ && right.Type() == object.STRUCT_OBJ:
		// only allowed operation is assign (=) by making the left have a copy of the value of the right side
		return i.evalAssignmentExpression(op, leftNode, left, right)

	case left.Type() == object.ARRAY_OBJ && right.Type() == object.ARRAY_OBJ:
		// only allowed for assign operator
		return i.evalAssignmentExpression(op, leftNode, left, right)

	case left.Type() == object.MAP_OBJ && right.Type() == object.MAP_OBJ:
		// only allowed for assign operator
		return i.evalAssignmentExpression(op, leftNode, left, right)

	case left.Type() != right.Type():
		return newError(ERROR, "type mismatch: %s %s %s",
			left.Type(), op, right.Type())

	default:
		return newError(ERROR, "unknown operator: %s %s %s",
			left.Type(), op, right.Type())
	}

}

func (i *Interpreter) evalIntegerInfixExpression(op string, lt, rt object.Object) object.Object {
	l, leftMutable := object.Cast(lt)
	r, _ := object.Cast(rt)

	// cast them to integers
	left := l.(*object.Integer)
	right := r.(*object.Integer)

	switch op {
	// arithmetic operations
	case lexer.TokenMultiply:
		return &object.Integer{
			Value: left.Value * right.Value,
		}
	case lexer.TokenSlash:
		// needs to be this way so the output of math operations like pow work properly
		res := float64(left.Value) / float64(right.Value)
		return &object.Float{
			Value: res,
		}
	case lexer.TokenPlus:
		return &object.Integer{
			Value: left.Value + right.Value,
		}
	case lexer.TokenMinus:
		return &object.Integer{
			Value: left.Value - right.Value,
		}

		// comparison operators
	case lexer.TokenGreater:
		return nativeBooleanObject(left.Value > right.Value)
	case lexer.TokenGreaterOrEqual:
		return nativeBooleanObject(left.Value >= right.Value)
	case lexer.TokenLess:
		return nativeBooleanObject(left.Value < right.Value)
	case lexer.TokenLessOrEqual:
		return nativeBooleanObject(left.Value <= right.Value)
	case lexer.TokenNotEquals:
		return nativeBooleanObject(left.Value != right.Value)
	case lexer.TokenEquals:
		return nativeBooleanObject(left.Value == right.Value)

	case lexer.TokenAssign:
		if leftMutable {
			left.Value = right.Value
			return left
		} else {
			// error saying this can't be mutable
			return newError(ERROR, "%v can't be mutate, since it was defined as const", left)
		}

	default:
		return newError(ERROR, "unknown operator: %s %s %s",
			left.Type(), op, right.Type())

	}

}

func (i *Interpreter) evalFloatInfixExpression(op string, lt, rt object.Object) object.Object {

	l, leftMutable := object.Cast(lt)
	r, _ := object.Cast(rt)

	// cast them to floats
	left := l.(*object.Float)
	right := r.(*object.Float)

	switch op {
	case lexer.TokenMultiply:
		return &object.Float{
			Value: left.Value * right.Value,
		}
	case lexer.TokenSlash:
		return &object.Float{
			Value: left.Value / right.Value,
		}
	case lexer.TokenPlus:
		return &object.Float{
			Value: left.Value + right.Value,
		}
	case lexer.TokenMinus:
		return &object.Float{
			Value: left.Value - right.Value,
		}

	case lexer.TokenGreater:
		return nativeBooleanObject(left.Value > right.Value)
	case lexer.TokenGreaterOrEqual:
		return nativeBooleanObject(left.Value >= right.Value)
	case lexer.TokenLess:
		return nativeBooleanObject(left.Value < right.Value)
	case lexer.TokenLessOrEqual:
		return nativeBooleanObject(left.Value <= right.Value)
	case lexer.TokenNotEquals:
		return nativeBooleanObject(left.Value != right.Value)
	case lexer.TokenEquals:
		return nativeBooleanObject(left.Value == right.Value)
	case lexer.TokenAssign:
		if leftMutable {
			left.Value = right.Value
			return left
		} else {
			// error saying this can't be mutable
			return newError(ERROR, "%v can't be mutate, since it was defined as const", left)
		}

	default:
		return newError(ERROR, "unknown operator: %s %s %s",
			left.Type(), op, right.Type())
	}

}

func (i *Interpreter) evalBooleanInfixExpression(op string, lt, rt object.Object) object.Object {

	l, leftMutable := object.Cast(lt)
	r, _ := object.Cast(rt)

	// cast them to booleans
	left := l.(*object.Boolean)
	right := r.(*object.Boolean)

	switch op {
	case lexer.TokenEquals:
		return nativeBooleanObject(left.Value == right.Value)
	case lexer.TokenNotEquals:
		return nativeBooleanObject(left.Value != right.Value)
	case lexer.TokenAnd:
		return nativeBooleanObject(left.Value && right.Value)
	case lexer.TokenOr:
		return nativeBooleanObject(left.Value || right.Value)

	case lexer.TokenAssign:
		if leftMutable {
			left.Value = right.Value
			return left
		} else {
			// error saying this can't be mutable
			return newError(ERROR, "%v can't be mutate, since it was defined as const", left)
		}

	default:
		// error
		return newError(ERROR, "Unsupported operator: %s %s %s",
			left.Type(), op, right.Type())
	}
}

func (i *Interpreter) evalStringInfixExpression(op string, left, right object.Object) object.Object {

	switch op {
	case lexer.TokenPlus:
		// cool do the concat
		return &object.String{
			Value: left.Inspect() + right.Inspect(),
		}

	// comparison
	case lexer.TokenEquals:
		return &object.Boolean{
			Value: left.Inspect() == right.Inspect(),
		}
	case lexer.TokenNotEquals:
		return &object.Boolean{
			Value: left.Inspect() != right.Inspect(),
		}

	// value assign
	case lexer.TokenAssign:
		// cast of the left side
		l, leftMutable := object.Cast(left)

		if !leftMutable {
			return newError(ERROR, "%v can't be mutate, since it was defined as const", left)
		}

		r, _ := object.Cast(right)

		switch left := l.(type) {
		case *object.String:
			// associate the right if it gets evaluated to a string
			if right, ok := r.(*object.String); ok {
				left.Value = right.Value
				return left
			}

			return newError(ERROR, "Can't assign a value of different type %s, %s",
				left.Type(), right.Type())

		default:
			// this checks for other values check when the get a string as right side
			return newError(ERROR, "Can't assign a value of different type %s, %s",
				left.Type(), right.Type())
		}
	}

	return newError(ERROR, "Unsupported operator: %s %s %s",
		left.Type(), op, right.Type())
}

func (i *Interpreter) areStructsCompatible(left, right *object.Struct) bool {
	if len(left.Fields) != len(right.Fields) {
		return false
	}

	for fieldName, leftField := range left.Fields {
		rightField, exists := right.Fields[fieldName]
		if !exists {
			return false
		}

		if leftField.Type() != rightField.Type() {
			return false
		}
	}

	return true
}

func (i *Interpreter) areArraysCompatible(left, right *object.Array) bool {
	if len(left.Elements) == 0 || len(right.Elements) == 0 {
		return true
	}

	return left.Elements[0].Type() == right.Elements[0].Type()
}

func (i *Interpreter) areMapsCompatible(left, right *object.Map) bool {
	if len(left.Pairs) == 0 || len(right.Pairs) == 0 {
		return true
	}

	var leftKeyType, leftValueType, rightKeyType, rightValueType object.ObjectType

	for _, leftPair := range left.Pairs {
		leftKeyType = leftPair.Key.Type()
		leftValueType = leftPair.Value.Type()
		// one operation only needed
		break
	}

	for _, rightPair := range right.Pairs {
		rightKeyType = rightPair.Key.Type()
		rightValueType = rightPair.Value.Type()
		// one operation only needed
		break
	}

	return leftKeyType == rightKeyType && leftValueType == rightValueType
}

// this function is responsible for handling assign op for both struct, hashmaps, structs
// Note: the assignment does a shallow copy, so modifying the value here will modify will affect the right struct instance
// for deep copy, there is copy function in the builtin module of stdlib that allows u todo that
func (i *Interpreter) evalAssignmentExpression(op string, leftNode ast.Expression, left, right object.Object) object.Object {
	if op != lexer.TokenAssign {
		return newError(ERROR, "operation with %s operator isn't allowed on type %s", op, left.Type())
	}

	if identifier, ok := leftNode.(*ast.Identifier); !ok {
		return newError(ERROR, "left side of assignment operation needs to be an identifier ")
	} else {
		// no need to check for mutability since array/hashmaps aren't allowed to be constants
		lft, leftMutable := object.Cast(left)
		lrt, _ := object.Cast(right)

		typeCheck := false
		errMsg := ""
		switch lft := lft.(type) {
		case *object.Struct:
			rt := lrt.(*object.Struct)
			typeCheck = i.areStructsCompatible(lft, rt)
			errMsg = "type mismatch on struct elements"

		case *object.Array:
			rt := lrt.(*object.Array)
			typeCheck = i.areArraysCompatible(lft, rt)
			errMsg = "type mismatch on array elements"

		case *object.Map:
			rt := lrt.(*object.Map)
			typeCheck = i.areMapsCompatible(lft, rt)
			errMsg = "type mismatch on map elements"
		}

		if !typeCheck {
			return newError(ERROR, errMsg)
		}

		// build a method into the env, and update it to left side
		i.env.OverrideDefine(identifier.Value, object.ItemObject{
			Object:    lrt,
			IsMutable: leftMutable,
		})
		return lrt
	}
}
func (i *Interpreter) evalMembershipExpression(owner object.Object, obj, property ast.Expression) object.Object {
	// switch on the object after cast

	owner, _ = object.Cast(owner)

	switch owner := owner.(type) {
	case *object.BuiltInModule:
		// evaluate the property
		switch ownerProperty := property.(type) {
		case *ast.CallExpression:
			// search for the corresponding property call and invoke
			function, ok := owner.Attrs[ownerProperty.Function.Value]
			if !ok {
				return newError(ERROR, "function doesn't exist on the module %s", owner.Name)
			}
			// invokes the call expression
			ableToCast := function.(object.ItemObject).IsBuiltIn
			args := i.evalExpressions(ownerProperty.Args, !ableToCast)
			if len(args) == 1 && isError(args[0]) {
				// error out
				return args[0]
			}
			return i.applyFunction(function, args)

		case *ast.Identifier:
			// a given constant in a module
			identifier, ok := owner.Attrs[ownerProperty.Value]
			if !ok {
				return newError(ERROR, "identifier doesn't exist on the module %s", ownerProperty.Value)
			}
			// no need for casting
			return identifier

		default:
			return newError(ERROR, "property needs to be of type call expression or identifier, for now")
		}

	case *object.Struct:
		switch ownerProperty := property.(type) {
		case *ast.CallExpression:
			// search for the corresponding property call and invoke
			methodItem, ok := owner.Methods[ownerProperty.Function.Value]
			if !ok {
				return newError(ERROR, "method doesn't exist on the struct %v", ownerProperty.Function)
			}

			// responsible to detect if the current accessed method is private or not
			// so private methods are only allowed within the struct scope methods
			if strings.HasPrefix(ownerProperty.Function.Value, "_") {
				if obj.GetToken().Text != lexer.TokenSelf {
					return newError(ERROR, "%s is a private method, u can't use outside of the struct", ownerProperty.Function.Value)
				}
			}

			// methodItem is object.ItemObject wrapping the *object.Function
			castFn, _ := object.Cast(methodItem)
			fn := castFn.(*object.Function)

			// Evaluate method args normally (do not include self yet)
			ableToCast := methodItem.(object.ItemObject).IsBuiltIn
			args := i.evalExpressions(ownerProperty.Args, !ableToCast)
			if len(args) == 1 && isError(args[0]) {
				return args[0]
			}

			// PREPEND the instance as the first argument (self)
			// Note: owner is already an object.Object (the struct instance)
			args = append([]object.Object{object.DeepCopy(owner)}, args...)

			// Now invoke the function using the normal applyFunction path.
			return i.applyFunction(fn, args)

		case *ast.Identifier:
			// a given constant in a module
			identifier, ok := owner.Fields[ownerProperty.Value]
			if !ok {
				return newError(ERROR, "identifier doesn't exist on the struct %v", obj)
			}

			// responsible to detect if the current accessed method is private or not
			// so private methods are only allowed within the struct scope methods
			if strings.HasPrefix(ownerProperty.Value, "_") {
				if obj.GetToken().Text != lexer.TokenSelf {
					return newError(ERROR, "%s is a private field, u can't use outside of the struct", ownerProperty.Value)
				}
			}

			// no need for casting
			return identifier

		case *ast.MemberShipExpression:

			// the immediate property here is going to be the new part owner of the other property that u want to get access to
			// example of this: self.person.greet()
			// self.person is going to become the immediate property
			// and the property is greet() method
			immediateProperty := i.evalMembershipExpression(owner, obj, ownerProperty.Object)
			if isError(immediateProperty) {
				return immediateProperty
			}

			// Then continue with the nested property
			return i.evalMembershipExpression(immediateProperty, ownerProperty, ownerProperty.Property)

		case *ast.IndexExpression:

			// handle array/map access on struct fields
			// get the field that contains the array/map
			fieldObj := i.evalMembershipExpression(owner, obj, ownerProperty.Left)
			if isError(fieldObj) {
				return fieldObj
			}

			// cast to get the actual object (unwrap ItemObject if needed)
			actualField, _ := object.Cast(fieldObj)

			// evaluate the index
			index := i.Eval(ownerProperty.Index)
			if isError(index) {
				return index
			}

			return i.evalIndexExpression(actualField, index)

		default:
			return newError(ERROR, "struct only support call expression, or identifier access, what u're doing isn't allowed")
		}

	default:
		return newError(ERROR, "Unsupported evaluation on this type: %s", owner.Type())
	}
}
