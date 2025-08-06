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
		// search for the module
		module, ok := stdlib.BuiltinModules[nd.ModuleName.Value]
		if !ok {
			return newError("Module Not found %s", nd.ModuleName)
		}
		for name, fn := range module {
			// means that this function is internal and can't be used
			// doesn't make so much sense cause u can just not register them
			if strings.HasPrefix(name, "_") {
				continue
			}
			// register function so u it can be used
			i.env.Define(name, object.ItemObject{
				Object:    fn,
				IsMutable: false,
				IsBuiltIn: true,
			})
		}

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

	case *ast.VarDeclaration:
		val := i.Eval(nd.Value)
		if isError(val) {
			return val
		}
		// define it in the scope
		if nd.Token.Text == "let" {
			i.env.Define(nd.Name.Value, object.ItemObject{
				Object:    val,
				IsMutable: true,
			})
		} else {
			// constant
			i.env.Define(nd.Name.Value, object.ItemObject{
				Object:    val,
				IsMutable: false,
			})
		}

	case *ast.Identifier:
		return i.evalIdentifier(nd)

	case *ast.FunctionExpression:
		params := nd.Args
		body := nd.Body
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
		return i.evalBlockStatement(nd.Body)

	case *ast.BlockStatement:
		return i.evalBlockStatement(nd)

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
		return buildInFunc
	}

	if builtInCons, ok := builtInConstants[identifier.Value]; ok {
		return builtInCons
	}

	return newError("identifier not found: %s", identifier.Value)
}

func (i *Interpreter) applyFunction(fn object.Object, args []object.Object) object.Object {

	//
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
		env.Define(param.Value, object.ItemObject{
			Object: args[paramIdx],
			// this makes the params mutable
			IsMutable: true,
		})
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
