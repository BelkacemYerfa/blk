package interpreter

import (
	"blk/ast"
	"blk/lexer"
	"blk/object"
	"blk/parser"
	"blk/stdlib"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	Skip  = &object.Skip{}
	Break = &object.Break{}
)

type Interpreter struct {
	env           *object.Environment
	cachedModules map[string]object.Object
	loadingMods   map[string]bool // tracks modules being loaded
	path          string
}

func NewInterpreter(env *object.Environment, path string) *Interpreter {
	if env == nil {
		env = object.NewEnvironment(nil)
	}
	loadingMods := make(map[string]bool)
	loadingMods[path] = true
	return &Interpreter{
		env:           env,
		cachedModules: make(map[string]object.Object),
		loadingMods:   loadingMods,
		path:          path,
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

type LeftRes struct {
	Object object.Object
	node   ast.Expression
}

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
		return i.evalModuleImport(nd)

	case *ast.StructExpression:
		methods := make(map[string]object.Object, 0)
		fields := make(map[string]object.Object, 0)

		// var declaration such as (x := 0, name :: "john")
		for _, decl := range nd.Fields {
			val := i.Eval(decl.Value)
			if isError(val) {
				return val
			}

			varDecl := object.ItemObject{
				// this works as the way done when declaring stuff, where
				Object:    object.UseCopyValueOrRef(val),
				IsMutable: decl.Token.Kind == lexer.TokenLet,
			}

			fields[decl.Name[0].Value] = varDecl
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
		structDef, ok := castDef.(*object.Struct)

		if !ok {
			return newError(ERROR, "to create a struct instance, u need to use a struct")
		}

		structDefCopy := structDef.Copy().(*object.Struct)
		copyOfStructDef := &object.StructInstance{
			Fields:  structDefCopy.Fields,
			Methods: structDef.Methods,
		}

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

			if fieldDef.Type() != fieldValue.Type() && fieldDef.Type() != object.NUL_OBJ {
				// type error
				return newError(ERROR, "type mismatch on %s, definition type %s, got %s", field.Key.Value, fieldDef.Type(), fieldValue.Type())
			}

			varDecl := object.ItemObject{
				// this works as the way done when declaring stuff, where
				Object:    object.UseCopyValueOrRef(fieldValue),
				IsMutable: true,
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
	case *ast.CharLiteral:
		return &object.Char{
			Value: nd.Value,
		}
	case *ast.BooleanLiteral:
		return nativeBooleanObject(nd.Value)
	case *ast.NulLiteral:
		return object.NUL

	case *ast.RangePattern:
		evalStart := i.Eval(nd.Start)
		if isError(evalStart) {
			return evalStart
		}

		if evalStart.Type() != object.INTEGER_OBJ {
			return newError(ERROR, "the left bound of range pattern needs to evaluate to a int, instead got %s", evalStart.Type())
		}

		evalEnd := i.Eval(nd.End)
		if isError(evalEnd) {
			return evalEnd
		}

		if evalEnd.Type() != object.INTEGER_OBJ {
			return newError(ERROR, "the right bound of range pattern needs to evaluate to a int, instead got %s", evalStart.Type())
		}

		// before .. token
		castedBound, _ := object.Cast(evalStart)
		leftBound := castedBound.(*object.Integer).Value
		// after .. token
		castedBound, _ = object.Cast(evalEnd)
		rightBound := castedBound.(*object.Integer).Value

		if leftBound > rightBound {
			return newError(ERROR, "the left bound can't be bigger than the right bound")
		}

		if len(nd.Op) > 0 && nd.Op == "=" {
			// this means the last element will be included
			// an example 1..9, will cover all elements from 1 to 8
			// 1..=9, will cover all elements from 1 to 9
			rightBound++
		}

		elements := []object.Object{}
		for i := range rightBound {
			if i < leftBound {
				continue
			}
			elements = append(elements, &object.Integer{Value: i})
		}

		return &object.Range{Elements: elements}

	case *ast.ArrayLiteral:
		elements := i.evalArrayExpression(nd.Elements)
		if len(elements) == 1 && isError(elements[0]) {
			return elements[0]
		}

		size := -1
		if nd.Size != nil {
			evaluatedSize := i.Eval(nd.Size)

			if isError(evaluatedSize) {
				return evaluatedSize
			}

			evaluatedSize, _ = object.Cast(evaluatedSize)
			if evaluatedSize.Type() != object.INTEGER_OBJ {
				return newError(ERROR, "size of an array needs to of type INTEGER, got %s", evaluatedSize.Type())
			}

			size = int(evaluatedSize.(*object.Integer).Value)

			// verify if the current size matches the amount of element inside of array if it is initialized
			if size < len(elements) {
				return newError(ERROR, "number of elements in the initialization exceeds the initial size, consider changing the size or adjusting the amount of elements")
			}
		}

		return &object.Array{Size: size, Elements: elements}

	case *ast.MapLiteral:
		return i.evalMapExpression(nd.Pairs)

	case *ast.IndexExpression:
		left := i.Eval(nd.Left)
		if isError(left) {
			return left
		}
		if !nd.Range {
			index := i.Eval(nd.Start)
			if isError(index) {
				return index
			}
			return i.evalIndexExpression(left, index)
		} else {
			if nd.Start == nil && nd.End == nil {
				return newError(ERROR, "can't use [i:j] syntax without providing at least one of the bound")
			}
			var Start, End object.Object
			// return an array from i to j
			if nd.Start != nil {
				Start = i.Eval(nd.Start)
				if isError(Start) {
					return Start
				}

				if Start.Type() != object.INTEGER_OBJ {
					return newError(ERROR, "the start bound needs to be of type integer, got %s", Start.Type())
				}
				Start, _ = object.Cast(Start)
			}

			if nd.End != nil {
				End = i.Eval(nd.End)
				if isError(End) {
					return End
				}

				if End.Type() != object.INTEGER_OBJ {
					return newError(ERROR, "the end bound needs to be of type integer, got %s", End.Type())
				}
				End, _ = object.Cast(End)
			}

			return i.evalRangeExpression(left, Start, End)
		}

	case *ast.WhileStatement:
		return i.evalWhileStatement(nd)

	case *ast.ForStatement:
		return i.evalForStatement(nd)

	case *ast.SkipStatement:
		return Skip

	case *ast.BreakStatement:
		return Break

	case *ast.VarDeclaration:
		val := i.Eval(nd.Value)
		if isError(val) {
			return val
		}

		return i.evalVarDeclaration(val, nd)

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
		returnValues := i.evalReturnValues(nd.ReturnValues)
		if len(returnValues) == 1 && isError(returnValues[0]) {
			return returnValues[0]
		}
		return &object.ReturnValue{Values: returnValues}

	case *ast.ScopeStatement:
		i.enterScope()
		defer i.exitScope()
		return i.evalBlockStatement(nd.Body)

	case *ast.BlockStatement:
		i.enterScope()
		defer i.exitScope()
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

		// For && and || operators, don't evaluate right side yet - let evalBinaryExpression handle it
		if nd.Operator == lexer.TokenAnd || nd.Operator == lexer.TokenOr {
			switch nd.Operator {
			case lexer.TokenAnd:
				if !object.IsTruthy(left) {
					return object.FALSE // short-circuit
				}
				r := i.Eval(nd.Right) // Only evaluate right side if left is truthy
				return nativeBooleanObject(object.IsTruthy(r))
			case lexer.TokenOr:
				if object.IsTruthy(left) {
					return object.TRUE // short-circuit
				}
				r := i.Eval(nd.Left) // Only evaluate right side if left is falsy
				return nativeBooleanObject(object.IsTruthy(r))
			}
		}

		right := i.Eval(nd.Right)
		if isError(right) {
			return right
		}

		return i.evalBinaryExpression(nd.Operator, left, right)

	case *ast.AssignStatement:
		leftResults := make([]LeftRes, 0)

		for _, left := range nd.Left {
			evaluated := i.Eval(left)
			if isError(evaluated) {
				return evaluated
			}
			leftResults = append(leftResults, LeftRes{
				Object: evaluated,
				node:   left,
			})
		}

		rightResults := make([]object.Object, 0)

		for _, right := range nd.Right {
			evaluated := i.Eval(right)
			if isError(evaluated) {
				return evaluated
			}

			castedVal, _ := object.Cast(evaluated)

			if castedVal.Type() == object.RETURN_VALUE_OBJ {
				returnValues := castedVal.(*object.ReturnValue).Values
				rightResults = append(rightResults, returnValues...)
			} else {
				rightResults = append(rightResults, evaluated)
			}
		}

		// this means there are more declaration than the assignments
		if len(leftResults) > len(rightResults) {
			return newError(ERROR, "found more identifiers than result values, consider adjusting either identifiers, or result values")
		}

		// otherwise we're cool, but still need to check the assignments
		return i.evalAssignment(leftResults, rightResults)

	case *ast.MemberShipExpression:
		// evaluate the owner
		obj := i.Eval(nd.Object)
		if isError(obj) {
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
			return res
		case *object.Error:
			return result
		}
	}

	return result
}

func (i *Interpreter) evalModuleImport(nd *ast.ImportStatement) object.Object {

	isModuleAPath := strings.Contains(nd.ModuleName.Value, "/")

	moduleName := nd.ModuleName.Value

	// if the module was saved with the alias name
	if nd.Alias != nil {
		moduleName = nd.Alias.Value
	}

	if module, ok := i.cachedModules[moduleName]; ok {
		return module
	}

	if isModuleAPath {

		cwd, _ := os.Getwd()
		cwd = filepath.Join(cwd, nd.ModuleName.Value)

		// means that the module is builtin into the std

		// cycle detection
		_, ok := i.loadingMods[cwd]
		if ok {
			moduleName, _ := os.Stat(i.path)
			circularModule, _ := os.Stat(cwd)
			return newError(ERROR, "circular dependency detected in module: %s, issue on %s import", moduleName.Name(), circularModule.Name())
		}

		i.loadingMods[cwd] = true
		defer func() { i.loadingMods[cwd] = false }()

		content, err := os.ReadFile(cwd)
		if err != nil {
			return newError(ERROR, err.Error())
		}
		l := lexer.NewLexer(cwd, string(content))
		p := parser.NewParser(l.Tokenize(), cwd)
		program := p.Parse()

		tempEnv := object.NewEnvironment(nil)

		moduleInterpreter := &Interpreter{
			env:           tempEnv,
			cachedModules: make(map[string]object.Object),
			loadingMods:   i.loadingMods,
			path:          cwd,
		}

		moduleEval := moduleInterpreter.Eval(program)

		// check if the eval triggers any errors on imported module
		if isError(moduleEval) {
			return moduleEval
		}

		exports := make(map[string]object.Object)
		for name, obj := range tempEnv.GetStore() {
			// skip private imports
			if strings.HasPrefix(name, "_") {
				continue
			}
			// save the module as ItemObject type
			exports[name] = obj
		}

		newModule := object.ItemObject{
			Object: &object.UserModule{
				Name:  nd.ModuleName.Value,
				Attrs: exports,
			},
			IsBuiltIn: true,
		}

		i.cachedModules[moduleName] = newModule
		i.env.Define(moduleName, newModule)

		return nil
	}

	module, ok := stdlib.BuiltinModules[nd.ModuleName.Value]
	if !ok && !isModuleAPath {
		return newError(ERROR, "Module Not found %s", nd.ModuleName)
	}

	newModule := object.ItemObject{
		Object: &object.BuiltInModule{
			Name:  nd.ModuleName.Value,
			Attrs: module,
		},
		IsBuiltIn: true,
	}

	// cache it first
	i.cachedModules[moduleName] = newModule
	// define it in the current env
	i.env.Define(moduleName, newModule)

	return nil
}

func nativeBooleanObject(val bool) *object.Boolean {
	if val {
		return object.TRUE
	}
	return object.FALSE
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
			// this is being used from the internal stdlib functions in this lang
			// by passing a reference so the var will get updated
		default:
			// wrap it inside of ItemObject struct
			argEval = object.ItemObject{
				Object: ev.Copy(),
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
		if !object.ObjectTypesCheck(firstElem, elemEval) {
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

// this is used for evaluating array elements
func (i *Interpreter) evalReturnValues(exps []ast.Expression) []object.Object {
	result := make([]object.Object, 0, len(exps))
	for _, e := range exps {
		evaluated := i.Eval(e)
		if isError(evaluated) {
			return []object.Object{evaluated}
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
		if !object.ObjectTypesCheck(valEl, value) {
			return newError(ERROR, "multitude of types detected, value elements of a map should be of one type")
		}
		pairs[hashed] = object.HashPair{Key: key, Value: value}
		idx++
	}

	return &object.Map{Pairs: pairs}
}

func (i *Interpreter) evalIndexExpression(lf, index object.Object) object.Object {
	// make sure that the left is either a map or an array
	lf, isLeftMutable := object.Cast(lf)
	index, _ = object.Cast(index)

	switch lf := lf.(type) {
	case *object.Array:
		if index.Type() != object.INTEGER_OBJ {
			return newError(ERROR, "index side needs to be an integer, got %v", index.Type())
		}
		return i.evalArrayIndexExpression(lf, index)

	case *object.String:
		if index.Type() != object.INTEGER_OBJ {
			return newError(ERROR, "index side needs to be an integer, got %v", index.Type())
		}
		array := make([]object.Object, 0)
		for _, elem := range lf.Value {
			array = append(array, object.ItemObject{
				Object: &object.Char{
					Value: elem,
				},
				IsMutable: isLeftMutable,
			})
		}
		return i.evalArrayIndexExpression(&object.Array{
			Elements: array,
		}, index)

	case *object.Map:
		return i.evalMapIndexExpression(lf, index)

	default:
		return newError(ERROR, "left side needs to be either an array or map, got %v", lf.Type())
	}
}

func (i *Interpreter) evalRangeExpression(lf, start, end object.Object) object.Object {
	// make sure that the left is either a map or an array

	if lf.Type() != object.ARRAY_OBJ && lf.Type() != object.STRING_OBJ {
		return newError(ERROR, "left side needs to be either an array with range expression, got %v", lf.Type())
	}

	lf, _ = object.Cast(lf)

	// handle string case
	var left *object.Array
	if lf.Type() == object.STRING_OBJ {
		array := make([]object.Object, 0)
		for _, elem := range lf.(*object.String).Value {
			array = append(array, &object.Char{
				Value: elem,
			})
		}
		left = &object.Array{
			Elements: array,
		}
	} else {
		left = lf.(*object.Array)
	}

	// case of [i:j]
	if start != nil && end != nil {
		startVal := start.(*object.Integer).Value
		endVal := end.(*object.Integer).Value

		if startVal < 0 || endVal < 0 {
			return newError(ERROR, "both bound start, end should be greater >= 0")
		}

		// make sure that i <= j
		if startVal > endVal {
			return newError(ERROR, "the start bound should be less then the end bound, got %d>%d", startVal, endVal)
		}

		// check that the end bound <= len(left.Elements)
		if endVal > int64(len(left.Elements)) {
			return newError(ERROR, "the end bound should be less <= length of the array element bound")
		}

		return &object.Array{
			Elements: left.Elements[startVal:endVal],
		}
	}

	// case of [i:]
	if start != nil {
		startVal := start.(*object.Integer).Value
		if startVal < 0 {
			return newError(ERROR, "the start bound should be greater >= 0")
		}
		if startVal > int64(len(left.Elements)) {
			return newError(ERROR, "the start bound should be less <= length of the array element bound")
		}

		return &object.Array{
			Elements: left.Elements[startVal:],
		}
	}

	// case of [:j]
	if end != nil {
		endVal := end.(*object.Integer).Value
		if endVal < 0 {
			return newError(ERROR, "the end bound should be greater >= 0")
		}
		if endVal > int64(len(left.Elements)) {
			return newError(ERROR, "the end bound should be less <= length of the array element bound")
		}

		return &object.Array{
			Elements: left.Elements[:endVal],
		}
	}

	return newError(ERROR, "weird case, not counted for")
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

func (i *Interpreter) evalVarDeclaration(val object.Object, nd *ast.VarDeclaration) object.Object {
	// this tells the interpreter that those type of values aren't allowed to be const
	if val.Type() == object.NUL_OBJ && !nd.Mutable {
		return newError(ERROR, "%v isn't allowed to be an const, consts are only: ints, floats, strings, booleans", val.Type())
	}

	// functions need to be declared as consts
	if (val.Type() == object.FUNCTION_OBJ) && nd.Mutable {
		return newError(ERROR, "functions are required to be declared as consts")
	}

	castedVal, _ := object.Cast(val)

	switch v := castedVal.(type) {
	case *object.Array:
		// if it is a const mark all of the values as const
		if !nd.Mutable {
			for idx, elem := range v.Elements {
				elem, _ = object.Cast(elem)
				v.Elements[idx] = object.ItemObject{
					Object: elem,
				}
			}
		}
	case *object.Map:
		// if it is a const mark all of the values as const
		if !nd.Mutable {
			for key, val := range v.Pairs {
				elem, _ := object.Cast(val.Value)
				v.Pairs[key] = object.HashPair{
					Key: val.Key,
					Value: object.ItemObject{
						Object: elem,
					},
				}
			}
		}
	}

	newVal := object.ItemObject{
		Object:    object.UseCopyValueOrRef(val),
		IsMutable: nd.Mutable,
	}

	// for multi value assignment from functions
	if castedVal.Type() == object.RETURN_VALUE_OBJ {
		returnValues := castedVal.(*object.ReturnValue).Values
		// the rule here is that var declaration need to be <= len(returnValue) elements
		if len(nd.Name) > len(returnValues) {
			return newError(ERROR, "numbers of declaration need to be less or equals to the number of return values")
		}

		// this handles the declaration of multi values
		for idx, ident := range nd.Name {
			currentVarAssigned := object.ItemObject{
				Object:    returnValues[idx].Copy(),
				IsMutable: newVal.IsMutable,
			}
			// define it in the scope
			_, firstDeclaration := i.env.Define(ident.Value, currentVarAssigned)

			if firstDeclaration {
				return newError(WARNING, "name %s is already in use", ident.Value)
			}
		}
	} else {
		singleVar := nd.Name[0]
		// define it in the scope
		_, firstDeclaration := i.env.Define(singleVar.Value, newVal)

		if firstDeclaration {
			return newError(WARNING, "name %s is already in use", singleVar.Value)
		}
	}

	return nil
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
				Object:    args[0],
				IsMutable: true,
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
		if len(returnValue.Values) <= 1 {
			return returnValue.Values[0]
		}
		return returnValue
	}
	return obj
}

func (i *Interpreter) evalBlockStatement(block *ast.BlockStatement) object.Object {
	var result object.Object

	for _, statement := range block.Body {
		result = i.Eval(statement)
		if result != nil {
			rt := result.Type()
			if rt == object.RETURN_VALUE_OBJ || rt == object.ERROR_OBJ || rt == object.BREAK_OBJ || rt == object.SKIP_OBJ {
				return result
			}
		}
	}

	return result
}

func (i *Interpreter) evalForStatement(nd *ast.ForStatement) object.Object {
	// check that the target is either an array or a map
	target, _ := object.Cast(i.Eval(nd.Target))

	if isError(target) {
		return target
	}

	iterable, ok := target.(object.Iterable)

	if !ok {
		return newError(ERROR, "target needs to be either an array or a map, got %s", target.Type())
	}

	items := iterable.Iter()
	if len(items) == 0 {
		return nil
	}

	i.enterScope()
	defer i.exitScope()

	for _, item := range items {
		// bind identifiers
		if len(nd.Identifiers) >= 1 && nd.Identifiers[0].Value != "_" {
			if target.Type() == object.RANGE_OBJ {
				i.env.OverrideDefine(nd.Identifiers[0].Value, object.ItemObject{Object: item.Value})
			} else {
				i.env.OverrideDefine(nd.Identifiers[0].Value, object.ItemObject{Object: item.Index})
			}
		}

		if target.Type() != object.RANGE_OBJ {
			if len(nd.Identifiers) >= 2 && nd.Identifiers[1].Value != "_" {
				i.env.OverrideDefine(nd.Identifiers[1].Value, object.ItemObject{Object: item.Value})
			}
		}

		// evaluate body
		res := i.Eval(nd.Body)
		if res != nil {
			switch res.Type() {
			case object.RETURN_VALUE_OBJ:
				return res
			case object.SKIP_OBJ:
				continue
			case object.BREAK_OBJ:
				return nil
			case object.ERROR_OBJ:
				return res
			}
		}
	}

	return nil
}

func (i *Interpreter) evalWhileStatement(nd *ast.WhileStatement) object.Object {
	condition := i.Eval(nd.Condition)

	if isError(condition) {
		return condition
	}

	if condition.Type() != object.BOOLEAN_OBJ && condition.Type() != object.NUL_OBJ {
		return newError(ERROR, "evaluation of the condition in a while loop needs to return a boolean not %s", condition)
	}

	condition, _ = object.Cast(condition)

	for object.IsTruthy(condition) {
		res := i.Eval(nd.Body)
		if res != nil {
			switch res.Type() {
			case object.RETURN_VALUE_OBJ:
				return res
			case object.SKIP_OBJ:
				continue
			case object.BREAK_OBJ:
				// break out of the loop
				break
			case object.ERROR_OBJ:
				return res
			}
		}

		condition = i.Eval(nd.Condition)
		if isError(condition) {
			return condition
		}
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
		}
		// eval the alternative
		return i.Eval(nd.Alternative)
	case *object.Nul:
		// check of nul
		return i.Eval(nd.Alternative)
	default:
		// error out

		return newError(ERROR, "evaluation of the condition needs to return a boolean not %s", cdn)
	}
}

func (i *Interpreter) evalUnaryExpression(op string, right object.Object) object.Object {
	switch op {
	case lexer.TokenExclamation:
		// check the right side
		// nul operator introduces a falsy mechanism to add
		if right.Type() == object.BOOLEAN_OBJ || right.Type() == object.NUL_OBJ {
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

	switch rt := rt.(type) {
	case *object.Nul:
		return object.TRUE
	case *object.Boolean:
		if rt.Value {
			return object.FALSE
		}

		return object.TRUE
	}

	return nil
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

func (i *Interpreter) evalAssignment(left []LeftRes, right []object.Object) object.Object {
	var result object.Object // Track the last assignment result

	for idx, leftRes := range left {
		if idx >= len(right) {
			return newError(ERROR, "not enough values to assign")
		}

		rightVal := right[idx]
		node := leftRes.node
		leftObj, leftMutable := object.Cast(leftRes.Object)
		rightObj, _ := object.Cast(rightVal)

		// Check mutability first
		if !leftMutable {
			return newError(ERROR, "%v can't be mutated, since it was defined as const", leftObj.Inspect())
		}

		// Type compatibility check
		// for nul value, u can assign it with what u want, then u need to respect the type rule that you're going to have
		// a value can be nullified if it has a certain value attached to it whatever the value is
		if leftObj.Type() != rightObj.Type() && leftObj.Type() != object.NUL_OBJ && rightObj.Type() != object.NUL_OBJ {
			return newError(ERROR, "type mismatch: can't assign %s to %s",
				rightObj.Type(), leftObj.Type())
		}

		// Perform the assignment based on type
		switch leftTyped := leftObj.(type) {
		case *object.Nul:
			switch node := node.(type) {
			case *ast.Identifier:
				i.env.OverrideDefine(node.Value, object.ItemObject{
					Object:    object.UseCopyValueOrRef(rightObj),
					IsMutable: leftMutable,
				})

			case *ast.MemberShipExpression:
				evalObj := i.Eval(node.Object)
				if isError(evalObj) {
					return evalObj
				}

				return i.evalRecursiveAssignment(evalObj, rightObj, node.Object, node.Property)

			default:
				return newError(ERROR, "left side of assignment operation needs to be an identifier ")
			}

			return rightObj

		case *object.Integer:
			if rightTyped, ok := rightObj.(*object.Integer); ok {
				leftTyped.Value = rightTyped.Value
				result = leftTyped
			}
		case *object.Float:
			if rightTyped, ok := rightObj.(*object.Float); ok {
				leftTyped.Value = rightTyped.Value
				result = leftTyped
			}
		case *object.Boolean:
			if rightTyped, ok := rightObj.(*object.Boolean); ok {
				leftTyped.Value = rightTyped.Value
				result = leftTyped
			}
		case *object.String:
			if rightTyped, ok := rightObj.(*object.String); ok {
				leftTyped.Value = rightTyped.Value
				result = leftTyped
			}
		case *object.Char:
			if rightTyped, ok := rightObj.(*object.Char); ok {
				if node, ok := node.(*ast.IndexExpression); ok {
					// left side is an array
					eval := i.Eval(node.Left)
					if eval.Type() == object.STRING_OBJ {
						return newError(ERROR, "left side is a string, for that index is not assignable")
					}
				}

				leftTyped.Value = rightTyped.Value
				result = leftTyped
			}
		case *object.StructInstance:
			switch node := node.(type) {
			case *ast.Identifier:
				result = i.evalAssignmentExpression(node, leftObj, rightObj)
				if isError(result) {
					return result
				}

			case *ast.MemberShipExpression:
				evalObj := i.Eval(node.Object)
				if isError(evalObj) {
					return evalObj
				}

				return i.evalRecursiveAssignment(evalObj, rightObj, node.Object, node.Property)

			default:
				return newError(ERROR, "left side of assignment operation needs to be an identifier ")
			}
		case *object.Array, *object.Map:
			// For complex types, delegate to existing method
			result = i.evalAssignmentExpression(node, leftObj, rightObj)
			if isError(result) {
				return result
			}
		default:
			return newError(ERROR, "unsupported assignment for type %s", leftObj.Type())
		}
	}

	return result // Return the result of the last assignment
}

func (i *Interpreter) evalBinaryExpression(op string, left, right object.Object) object.Object {
	left, _ = object.Cast(left)
	right, _ = object.Cast(right)

	// Check if either operand is a type that doesn't support binary operations
	if left.Type() == object.ARRAY_OBJ || left.Type() == object.MAP_OBJ ||
		right.Type() == object.ARRAY_OBJ || right.Type() == object.MAP_OBJ ||
		left.Type() == object.STRUCT_OBJ || right.Type() == object.STRUCT_OBJ ||
		left.Type() == object.FUNCTION_OBJ || right.Type() == object.FUNCTION_OBJ {
		return newError(ERROR, "binary operations not supported on types: %s %s %s",
			left.Type(), op, right.Type())
	}

	return left.Binary(op, right)
}

// this function is responsible for handling assign op for both struct, hashmaps, structs
// Note: the assignment does a shallow copy, so modifying the value here will modify will affect the right struct instance
// for deep copy, there is copy function in the builtin module of stdlib that allows u todo that
func (i *Interpreter) evalAssignmentExpression(leftNode ast.Expression, left, right object.Object) object.Object {

	identifier, ok := leftNode.(*ast.Identifier)

	if !ok {
		return newError(ERROR, "left side of assignment operation needs to be an identifier ")
	}

	// mutability already checked in the evalAssignment function, no need to repeat it here
	lft, leftMutable := object.Cast(left)
	lrt, _ := object.Cast(right)

	errMsg := ""
	switch lft.(type) {
	case *object.Array:
		errMsg = "type mismatch on array elements"

	case *object.Map:
		errMsg = "type mismatch on map elements"
	}

	typeCheck := object.ObjectTypesCheck(lft, lrt)

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
			ableToCast := true
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

	case *object.UserModule:
		switch ownerProperty := property.(type) {
		case *ast.CallExpression:
			// search for the corresponding property call and invoke
			function, ok := owner.Attrs[ownerProperty.Function.Value]
			if !ok {
				return newError(ERROR, "function doesn't exist on the module %s", owner.Name)
			}
			// invokes the call expression
			// ableToCast := true
			args := i.evalExpressions(ownerProperty.Args, true)
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

	case *object.StructInstance:

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
			args = append([]object.Object{(owner)}, args...)

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

			if !ownerProperty.Range {
				// evaluate the index
				index := i.Eval(ownerProperty.Start)
				if isError(index) {
					return index
				}

				return i.evalIndexExpression(actualField, index)
			} else {
				if ownerProperty.Start == nil && ownerProperty.End == nil {
					return newError(ERROR, "can't use [i:j] syntax without providing at least one of the bound")
				}
				var Start, End object.Object
				// return an array from i to j
				if ownerProperty.Start != nil {
					Start = i.Eval(ownerProperty.Start)
					if isError(Start) {
						return Start
					}

					if Start.Type() != object.INTEGER_OBJ {
						return newError(ERROR, "the start bound needs to be of type integer, got %s", Start.Type())
					}
					Start, _ = object.Cast(Start)
				}

				if ownerProperty.End != nil {
					End = i.Eval(ownerProperty.End)
					if isError(End) {
						return End
					}

					if End.Type() != object.INTEGER_OBJ {
						return newError(ERROR, "the end bound needs to be of type integer, got %s", End.Type())
					}
					End, _ = object.Cast(End)
				}

				return i.evalRangeExpression(actualField, Start, End)
			}

		default:
			return newError(ERROR, "struct only support call expression, or identifier access, what u're doing isn't allowed")
		}

	default:
		return newError(ERROR, "Unsupported evaluation on this type: %s", owner.Type())
	}
}

func (i *Interpreter) evalRecursiveAssignment(ownerObj, rightObj object.Object, obj, property ast.Expression) object.Object {

	switch property := property.(type) {
	case *ast.Identifier:
		// Simple property access: obj.prop
		if ownerObj.Type() != object.STRUCT_INSTANCE_OBJ && ownerObj.Type() != object.STRUCT_OBJ {
			return newError(ERROR, "Unsupported evaluation on this type: %s", ownerObj.Type())
		}

		var fields map[string]object.Object

		castedOwner, _ := object.Cast(ownerObj)

		switch owner := castedOwner.(type) {
		case *object.Struct:
			fields = owner.Fields
		case *object.StructInstance:
			fields = owner.Fields
		}

		identifier, ok := fields[property.Value]
		if !ok {
			return newError(ERROR, "identifier doesn't exist on the struct %v", obj)
		}
		_, mutable := object.Cast(identifier)
		fields[property.Value] = object.ItemObject{
			Object:    rightObj,
			IsMutable: mutable,
		}
		return rightObj

	case *ast.IndexExpression:
		// handle array/map access on struct fields
		// get the field that contains the array/map
		fieldObj := i.evalMembershipExpression(ownerObj, obj, property.Left)
		if isError(fieldObj) {
			return fieldObj
		}

		// cast to get the actual object (unwrap ItemObject if needed)
		actualField, _ := object.Cast(fieldObj)

		if property.Range {
			return newError(ERROR, "can't assign values to a range")
		}

		// evaluate the index
		index := i.Eval(property.Start)
		if isError(index) {
			return index
		}

		index, _ = object.Cast(index)
		switch lf := actualField.(type) {
		case *object.Array:
			if index.Type() != object.INTEGER_OBJ {
				return newError(ERROR, "index side needs to be an integer, got %v", index.Type())
			}

			idx := index.(*object.Integer).Value
			max := int64(len(lf.Elements) - 1)

			if idx < 0 || idx > max {
				return newError(ERROR, "index out of bound, %d", idx)
			}

			lf.Elements[idx] = object.ItemObject{
				Object:    rightObj,
				IsMutable: true, // or get mutability from existing element
			}

			return rightObj

		case *object.Map:

			key, ok := index.(object.Hashable)
			if !ok {
				return newError(ERROR, "unusable as hash key: %s", index.Type())
			}

			pair, ok := lf.Pairs[key.HashKey()]

			if !ok {
				return newError(ERROR, "index (%v) is not associated with any value", index.Inspect())
			}

			lf.Pairs[key.HashKey()] = object.HashPair{
				Key:   pair.Key,
				Value: rightObj,
			}

			return rightObj

		default:
			return newError(ERROR, "left side needs to be either an array or map, got %v", lf.Type())
		}

	case *ast.MemberShipExpression:
		// Nested property: obj.prop1.prop2 or obj.prop1[0]

		// First get the intermediate object
		intermediateObj := i.evalMembershipExpression(ownerObj, obj, property.Object)
		if isError(intermediateObj) {
			return intermediateObj
		}

		// Cast to get the actual object if it's wrapped in ItemObject
		castedIntermediate, _ := object.Cast(intermediateObj)

		// Recursively assign to the intermediate object
		return i.evalRecursiveAssignment(castedIntermediate, rightObj, property.Object, property.Property)

	default:
		return newError(ERROR, "Unsupported property type in assignment: %T", property)
	}
}
