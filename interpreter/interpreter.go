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
		args := i.evalExpressions(nd.Args)
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

func (i *Interpreter) evalExpressions(exps []ast.Expression) []object.Object {
	var result []object.Object
	for _, e := range exps {
		evaluated := i.Eval(e)
		if isError(evaluated) {
			return []object.Object{evaluated}
		}
		argEval, _ := cast(evaluated)
		result = append(result, argEval)
	}
	return result
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
	fn, _ = cast(fn)

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
		fmt.Println("ERROR: ! operator can only be applied on boolean values")
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

	l, leftMutable := cast(lt)
	r, _ := cast(rt)
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

	l, leftMutable := cast(lt)
	r, _ := cast(rt)

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

	l, leftMutable := cast(lt)
	r, _ := cast(rt)

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

		l, leftMutable := cast(left)

		if !leftMutable {
			return newError("%v can't be mutate, since it was defined as const", left)
		}

		r, _ := cast(right)

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

func cast(obj object.Object) (object.Object, bool) {
	switch obj := obj.(type) {
	case object.ItemObject:
		return obj.Object, obj.IsMutable
	// case *object.ItemObject:
	// 	return obj.Object, obj.IsMutable
	default:
		return obj, false
	}
}
