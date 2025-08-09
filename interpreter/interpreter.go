package interpreter

import (
	"blk/ast"
	"blk/lexer"
	"blk/object"
	"blk/stdlib"
	"fmt"
	"reflect"
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

func newError(format string, a ...interface{}) *object.Error {
	return &object.Error{Message: fmt.Sprintf(format, a...)}
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
		// define the module name as identifier
		module, ok := stdlib.BuiltinModules[nd.ModuleName.Value]
		if !ok {
			return newError("Module Not found %s", nd.ModuleName)
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

		i.env.Define(nd.ModuleName.Value, object.ItemObject{
			Object: &object.BuiltInModule{
				Name:  nd.ModuleName.Value,
				Attrs: attrs,
			},
			IsBuiltIn: true,
		})

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
				return newError("methods of a struct can't be mutated")
			}

			fieldDef, ok := structDef.Fields[field.Key.Value]
			if !ok {
				return newError("%s field doesn't exist on the struct definition, consider declaring it", field.Key.Value)
			}

			fieldValue := i.Eval(field.Value)

			if fieldDef.Type() != fieldValue.Type() {
				// type error
				return newError("type mismatch on %s, definition type %s, got %s", field.Key.Value, fieldDef.Type(), fieldValue.Type())
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

	case *ast.VarDeclaration:
		val := i.Eval(nd.Value)
		if isError(val) {
			return val
		}

		switch v := val.(type) {
		// this is in case for the val is another var declaration
		// this ensures to make a copy of the value and not the
		case object.ItemObject:
			valueClone := object.DeepCopy(v.Object)
			newVal := object.ItemObject{
				Object: valueClone,
			}
			if nd.Token.Text == "let" {
				newVal.IsMutable = true
			}
			i.env.Define(nd.Name.Value, newVal)
		default:
			// define it in the scope
			newVal := object.ItemObject{
				Object: v,
			}
			if nd.Token.Text == "let" {
				newVal.IsMutable = true
			}
			i.env.Define(nd.Name.Value, newVal)
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
		i.enterScope()
		var evaluation object.Object
		// scope makes it easier to understand
		{
			evaluation = i.evalBlockStatement(nd.Body)
		}
		i.exitScope()
		return evaluation

	case *ast.BlockStatement:
		i.enterScope()
		var evaluation object.Object
		// scope makes it easier to understand
		{
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
		return i.evalBinaryExpression(nd.Operator, left, right)

	case *ast.MemberShipExpression:
		// evaluate the owner
		obj := i.Eval(nd.Object)
		if isError(obj) {
			fmt.Println(obj)
			return obj
		}

		return i.evalMembershipExpression(obj, nd.Property)

	default:
		fmt.Println("yehoo", reflect.TypeOf(nd))
	}
	return nil
}

func (i *Interpreter) evalProgram(stmts []ast.Statement) object.Object {
	var result object.Object
	for _, statement := range stmts {
		result = i.Eval(statement)

		switch res := result.(type) {
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
	} else {
		return FALSE
	}
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
		default:
			// wrap it inside of ItemObject struct
			argEval = object.ItemObject{
				Object: ev,
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
				newError("multitude of types, (%v,%v), array elements should be of one type", firstElem.Type(), elemEval.Type()),
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
			return newError("unusable as hash key: %s", key.Type())
		}
		if idx == 0 {
			keyEl = key
		}
		if keyEl.Type() != key.Type() {
			return newError("multitude of types, (%v,%v), key elements of a map should be of one type", keyEl.Type(), key.Type())
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
			return newError("multitude of types, (%v,%v), value elements of a map should be of one type", valEl.Type(), value.Type())
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
			return newError("index side needs to be an integer, got %v", index.Type())
		}

		return i.evalArrayIndexExpression(lf, index)
	case *object.Map:
		return i.evalMapIndexExpression(lf, index)

	default:
		return newError("left side needs to be either an array or map, got %v", lf.Type())
	}
}

func (i *Interpreter) evalArrayIndexExpression(array, index object.Object) object.Object {
	arrayObject := array.(*object.Array)
	idx := index.(*object.Integer).Value
	max := int64(len(arrayObject.Elements) - 1)

	if idx < 0 || idx > max {
		return newError("index out of bound, %d", idx)
	}

	return arrayObject.Elements[idx]
}

func (i *Interpreter) evalMapIndexExpression(hashMap, index object.Object) object.Object {
	mapObject := hashMap.(*object.Map)

	key, ok := index.(object.Hashable)
	if !ok {
		return newError("unusable as hash key: %s", index.Type())
	}

	pair, ok := mapObject.Pairs[key.HashKey()]
	if !ok {
		return newError("index (%v) is not associated with any value", index.Inspect())
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

	return newError("identifier not found: %s", identifier.Value)
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
			return newError("wrong number of arguments. got=%d, want=%d",
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
		return newError("not a function: %s", fn.Type())
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
	// restore the env
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
				for ix, ident := range nd.Identifiers {
					if ix == 0 {
						i.env.OverrideDefine(ident.Value, object.ItemObject{
							Object: &object.Integer{
								Value: int64(idx),
							},
						})
					}

					if ix == 1 {
						// get the a copy of the current element value
						newVal := object.DeepCopy(elem)
						i.env.OverrideDefine(ident.Value, object.ItemObject{
							Object: newVal,
						})
					}
				}
				// evaluate the body
				i.evalBlockStatement(nd.Body)
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
				for ix, ident := range nd.Identifiers {
					if ix == 0 {
						newVal := object.DeepCopy(elem.Key)
						i.env.OverrideDefine(ident.Value, object.ItemObject{
							Object: newVal,
						})
					}

					if ix == 1 {
						newVal := object.DeepCopy(elem.Value)
						i.env.OverrideDefine(ident.Value, object.ItemObject{
							Object: newVal,
						})
					}
				}
				// evaluate the body
				i.evalBlockStatement(nd.Body)
			}
		}
		i.exitScope()

	default:
		return newError("target needs to be either an array or a map, got %v", tg.Type())
	}

	return nil
}

func (i *Interpreter) evalWhileStatement(nd *ast.WhileStatement) object.Object {
	condition := i.Eval(nd.Condition)

	if isError(condition) {
		return condition
	}

	switch cdn := condition.(type) {
	case *object.Boolean:
		// continue until the condition is broken
		for cdn.Value {
			i.Eval(nd.Body)
			// ? need to check if the casting is cool maybe, not sure
			cdn = i.Eval(nd.Condition).(*object.Boolean)
		}
	default:
		// error out
		return newError("evaluation of the condition in a while loop needs to return a boolean not %s", cdn)
	}

	// maybe
	return nil
}

func (i *Interpreter) evalIfExpression(nd *ast.IfExpression) object.Object {
	condition := i.Eval(nd.Condition)

	if isError(condition) {
		return condition
	}

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
		return newError("ERROR: evaluation of the condition needs to return a boolean not %s", cdn)
	}
}

func (i *Interpreter) evalUnaryExpression(op string, right object.Object) object.Object {
	switch op {
	case lexer.TokenExclamation:
		// check the right side
		if right.Type() == object.BOOLEAN_OBJ {
			return i.evalBangOperatorExpression(right.(*object.Boolean))
		}
		// throw an error
		return newError("! operator can only be applied on boolean values")
	case lexer.TokenMinus:
		// support for both ints and floats
		return i.evalMinusPrefixOperatorExpression(right)
	default:
	}

	return newError("unknown operator: %s%s", op, right.Type())
}

func (i *Interpreter) evalBangOperatorExpression(right *object.Boolean) *object.Boolean {
	if right.Value {
		return FALSE
	} else {
		return TRUE
	}
}

func (i *Interpreter) evalMinusPrefixOperatorExpression(right object.Object) object.Object {
	switch right := right.(type) {
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
		return newError("unknown operator: -%s", right.Type())
	}
}

func (i *Interpreter) evalBinaryExpression(op string, left, right object.Object) object.Object {
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

	case left.Type() != right.Type():
		return newError("type mismatch: %s %s %s",
			left.Type(), op, right.Type())

	default:
		return newError("unknown operator: %s %s %s",
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
			return newError("%v can't be mutate, since it was defined as const", left)
		}

	default:
		return newError("unknown operator: %s %s %s",
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
			return newError("%v can't be mutate, since it was defined as const", left)
		}

	default:
		return newError("unknown operator: %s %s %s",
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
			return newError("%v can't be mutate, since it was defined as const", left)
		}

	default:
		// error
		return newError("Unsupported operator: %s %s %s",
			left.Type(), op, right.Type())
	}
}

func (i *Interpreter) evalStringInfixExpression(op string, left, right object.Object) object.Object {

	switch op {
	case lexer.TokenPlus:
		// cool do the concat
		return &object.String{
			Value: left.Inspect() + " " + right.Inspect(),
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
		// cast for this also

		l, leftMutable := object.Cast(left)

		if !leftMutable {
			return newError("%v can't be mutate, since it was defined as const", left)
		}

		r, _ := object.Cast(right)

		switch left := l.(type) {
		case *object.String:
			// associate the right if it gets evaluated to a string
			if right, ok := r.(*object.String); ok {
				left.Value = right.Value
				return left
			}

			return newError("Can't assign a value of different type %s, %s",
				left.Type(), right.Type())

		default:
			// this checks for other values check when the get a string as right side
			return newError("Can't assign a value of different type %s, %s",
				left.Type(), right.Type())
		}
	}

	return newError("Unsupported operator: %s %s %s",
		left.Type(), op, right.Type())
}

func (i *Interpreter) evalMembershipExpression(owner object.Object, property ast.Expression) object.Object {
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
				return newError("function doesn't exist on the module %s", owner.Name)
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
				return newError("identifier doesn't exist on the module %s", ownerProperty.Value)
			}
			// no need for casting
			return identifier

		default:
			return newError("property needs to be of type call expression or identifier, for now")
		}

	// ? More testing needed
	case *object.Struct:
		switch ownerProperty := property.(type) {
		case *ast.CallExpression:
			// search for the corresponding property call and invoke
			methodItem, ok := owner.Methods[ownerProperty.Function.Value]
			if !ok {
				return newError("method doesn't exist on the struct %v", ownerProperty.Function)
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
			args = append([]object.Object{owner}, args...)

			// Now invoke the function using the normal applyFunction path.
			return i.applyFunction(fn, args)

		case *ast.Identifier:
			// a given constant in a module
			identifier, ok := owner.Fields[ownerProperty.Value]
			if !ok {
				return newError("identifier doesn't exist on the struct %v", ownerProperty)
			}
			// no need for casting
			return identifier

		case *ast.MemberShipExpression:

			// the immediate property here is going to be the new part owner of the other property that u want to get access to
			// example of this: self.person.greet()
			// self.person is going to become the immediate property
			// and the property is greet() method
			immediateProperty := i.evalMembershipExpression(owner, ownerProperty.Object)
			if isError(immediateProperty) {
				return immediateProperty
			}

			// Then continue with the nested property
			return i.evalMembershipExpression(immediateProperty, ownerProperty.Property)

		default:
			fmt.Println(ownerProperty)
			return newError("struct only support call expression, or identifier access, what u're doing isn't allowed")
		}

	default:
		return newError("Unsupported evaluation on this type: %s", owner.Type())
	}
}
