package semantic

import (
	"blk/ast"
	"blk/lexer"
	"errors"
	"fmt"
	"math"
)

// Typechecker implementation
// Responsible of providing an implementation of blk type checker that will be using bidirectional typechecking approach to verify & analyze the types

type TypeChecker struct {
	filePath string
	symtab   *SymbolTable
	errors   []error
}

func NewTypeChecker(filepath string, symtab *SymbolTable) *TypeChecker {
	return &TypeChecker{
		filePath: filepath,
		symtab:   symtab,
		errors:   make([]error, 0),
	}
}

func (tc *TypeChecker) add(err error) {
	tc.errors = append(tc.errors, err)
}

// TODO: make the error a single collector struct to use on each phase
func (tc *TypeChecker) error(tok lexer.Token, msg ...interface{}) error {
	errMsg := fmt.Sprintf("\033[1;90m%s:%d:%d:\033[0m ERROR: %s", tc.filePath, tok.Row, tok.Col, fmt.Sprint(msg...))

	return errors.New(errMsg)
}

type HighlightFlag string

const (
	Reset   HighlightFlag = "\033[0m"
	Red     HighlightFlag = "\033[31m"
	Green   HighlightFlag = "\033[32m"
	Yellow  HighlightFlag = "\033[33m"
	Blue    HighlightFlag = "\033[34m"
	Magenta HighlightFlag = "\033[35m"
	Cyan    HighlightFlag = "\033[36m"
	Gray    HighlightFlag = "\033[37m"
	White   HighlightFlag = "\033[97m"
)

func (tc *TypeChecker) highlight(msg any, flag HighlightFlag) string {
	nwMsg := ""

	switch flag {
	case Red:
		nwMsg = fmt.Sprintf("%v%v%v", Red, msg, Reset)
	case Green:
		nwMsg = fmt.Sprintf("%v%v%v", Green, msg, Reset)

	case Yellow:
		nwMsg = fmt.Sprintf("%v%v%v", Yellow, msg, Reset)

	case Blue:
		nwMsg = fmt.Sprintf("%v%v%v", Blue, msg, Reset)

	case Magenta:
		nwMsg = fmt.Sprintf("%v%v%v", Magenta, msg, Reset)

	case Cyan:
		nwMsg = fmt.Sprintf("%v%v%v", Cyan, msg, Reset)

	case Gray:
		nwMsg = fmt.Sprintf("%v%v%v", Gray, msg, Reset)

	case White:
		nwMsg = fmt.Sprintf("%v%v%v", White, msg, Reset)

	default:
		// reset
	}

	return nwMsg
}

func (tc *TypeChecker) check(program *ast.Program) {
	for _, stmt := range program.Statements {
		tc.checkStmt(stmt)
	}
}

func (tc *TypeChecker) checkStmt(stmt ast.Statement) {
	switch s := stmt.(type) {
	case *ast.VarDeclaration:
		tc.checkVarDecl(s)
	case *ast.ExpressionStatement:
		tc.inferExpr(s.Expression) // side-effect: ensure expr is valid
	}
}

func (tc *TypeChecker) checkVarDecl(d *ast.VarDeclaration) {
	isExplicit := d.Type != nil
	isInit := d.Value != nil

	name := d.Name[0].String()

	if isExplicit && !isInit {
		// default initialization in this case
		// all number types to 0
		// string to ""
		// pointer to nul
		// char to ''
		// bool to false

		d.Value = []ast.Expression{}

		switch d.Type.Type() {
		case ast.TypeString:
			d.Value = append(d.Value, &ast.StringLiteral{Token: d.Token})

		case ast.TypeBool:
			d.Value = append(d.Value, &ast.BooleanLiteral{Token: d.Token})

		case ast.TypeChar:
			d.Value = append(d.Value, &ast.CharLiteral{Token: d.Token})

		case ast.TypePointer:
			d.Value = append(d.Value, &ast.NulLiteral{Token: d.Token})

		case ast.TypeFloat32, ast.TypeFloat64:
			d.Value = append(d.Value, &ast.FloatLiteral{Token: d.Token})

		case ast.TypeArray:
			d.Value = append(d.Value, &ast.ArrayLiteral{Token: d.Token, Type: d.Type})

		case ast.TypeMap:
			d.Value = append(d.Value, &ast.MapLiteral{Token: d.Token, Type: d.Type})

		default:
			d.Value = append(d.Value, &ast.IntegerLiteral{Token: d.Token})

		}

		// update the symtab decl
		tc.symtab.CurrentScope.Update(name, &Symbol{
			Name:      name,
			Kind:      d.Type,
			IsMutable: d.Mutable,
			DeclNode:  d,
		})
		return
	}

	expr := d.Value[0]

	sym := tc.symtab.CurrentScope.Resolve(name)

	if sym == nil {
		tc.add(tc.error(d.Token, "variable with ", name, " wasn't found"))
		return
	}

	if isExplicit && isInit {
		// check the explicit type against the inferred type
		inferred := tc.inferExpr(expr)

		errMsg := fmt.Sprintf("type mismatch, explicit type %v doesn't match inferred type (%v), change the explicit type, or let the compiler infer it with := syntax", tc.highlight(d.Type, Red), tc.highlight(inferred, Green))

		if !tc.typesCompatible(d.Type, inferred) {
			tc.add(tc.error(expr.GetToken(), errMsg))
		}
	}

	if !isExplicit && isInit {
		// no explicit type, type will get inferred from the assign expression
		d.Type = tc.inferExpr(expr)
	}

}

func (tc *TypeChecker) inferExpr(expr ast.Expression) ast.Type {
	switch e := expr.(type) {

	case *ast.UnaryExpression:
		return tc.inferUnaryExprType(e)

	case *ast.BinaryExpression:
		return tc.inferBinaryExprType(e)

	case *ast.FunctionExpression:
		return tc.inferFunctionType(e)

	case *ast.CallExpression:
		fnType := tc.inferFunctionExpr(e)
		return tc.checkCallAgainst(fnType, e)

	case *ast.IntegerLiteral, *ast.StringLiteral, *ast.CharLiteral, *ast.NulLiteral, *ast.FloatLiteral, *ast.BooleanLiteral:
		return tc.inferPrimitiveType(e)

	case *ast.MapLiteral:
		return tc.inferMapType(e)

	case *ast.ArrayLiteral:
		return tc.inferArrayType(e)

	default:
		tc.add(tc.error(expr.GetToken(), "cannot infer type for expression"))
		return nil
	}

}

func (tc *TypeChecker) inferUnaryExprType(expr *ast.UnaryExpression) ast.Type {
	rightType := tc.inferExpr(expr.Right)

	switch expr.Operator {
	case "!":
		if rightType.Type() != ast.TypeBool {
			tc.add(fmt.Errorf("bang operator can only be used with boolean, but the received type is %s", rightType))
			return nil
		}

		return rightType

	case "~":
		if rightType.Type() > ast.TypeUInt64 {
			tc.add(fmt.Errorf("bitwise not operator ~ can only be applied on signed int or unsigned int, but the received type is %v", rightType))
			return nil
		}

		return rightType

	case "-":
		if rightType.Type() > ast.TypeFloat64 {
			tc.add(fmt.Errorf("- operator can only be applied either on signed int or unsigned int or floats, but the received type is %v", rightType))
			return nil
		}

		rightType.(*ast.PrimitiveType).Signed = !rightType.(*ast.PrimitiveType).Signed

		return rightType
	}

	return nil
}

func (tc *TypeChecker) inferBinaryExprType(expr *ast.BinaryExpression) ast.Type {
	leftType := tc.inferExpr(expr.Left)
	rightType := tc.inferExpr(expr.Right)

	switch lfp := leftType.(type) {
	case *ast.FunctionType:
		leftType = lfp.Return[0]
	}

	switch lrp := rightType.(type) {
	case *ast.FunctionType:
		leftType = lrp.Return[0]
	}

	if !tc.typesCompatible(leftType, rightType) {
		tc.add(tc.error(expr.Token, "sides of binary expression are not equals, left ", leftType, " right ", rightType))
		return nil
	}

	switch expr.Operator {

	case "+", "-", "*", "/", "%":
		// cases allowed: string (plus only) , int, uint, floats

		if leftType.Type() > ast.TypeString || rightType.Type() > ast.TypeString {
			tc.add(tc.error(expr.Token, "arithmetic operators can only be applied on (integers, unsigned integers, floats, strings), but the received type is ", rightType))
			return nil
		}

		switch leftType.Type() {
		case ast.TypeString:
			if expr.Operator != "+" {
				tc.add(tc.error(expr.Token, "the only arithmetic operation allowed on strings is +, but received ", expr.Operator, " as operator"))
				return nil
			}

			return leftType

		case ast.TypeFloat32, ast.TypeFloat64:
			if expr.Operator == "%" {
				tc.add(tc.error(expr.Token, "% (modules) operator isn't allowed on float, it is only allowed on ints"))
				return nil
			}

			return leftType

		default:
			return leftType

		}

	case "==", "!=":
		// cases allowed: bool, string, int, uint, floats

		if leftType.Type() > ast.TypeBool || rightType.Type() > ast.TypeBool {
			tc.add(tc.error(expr.Token, "arithmetic operators can only be applied on (integers, unsigned integers, floats, strings), but the received type is ", rightType))
			return nil
		}

		return &ast.PrimitiveType{Token: expr.Token, Kind: ast.TypeBool}

	case "<", ">", "<=", ">=":
		// cases allowed: int, uint, floats

		if leftType.Type() > ast.TypeBool || rightType.Type() > ast.TypeBool {
			tc.add(tc.error(expr.Token, "comparison operators can only be applied on (integers, unsigned integers, floats, strings), but the received type is ", rightType))
			return nil
		}

		return &ast.PrimitiveType{Token: expr.Token, Kind: ast.TypeBool}

	case "||", "&&":
		//cases allowed: booleans

		if leftType.Type() != ast.TypeBool || rightType.Type() != ast.TypeBool {
			tc.add(tc.error(expr.Token, "arithmetic operators can only be applied on (integers, unsigned integers, floats, strings), but the received type is ", rightType))
			return nil
		}

		return &ast.PrimitiveType{Token: expr.Token, Kind: ast.TypeBool}

	case "<<", ">>", "^", "&", "|":
		// cases allowed int, uint
		if leftType.Type() > ast.TypeUInt64 || rightType.Type() > ast.TypeUInt64 {
			tc.add(tc.error(expr.Token, "bitwise operators can only be applied on (integers, unsigned integers), but the received type is ", rightType))
			return nil
		}

		return leftType

	}

	return leftType
}

func (tc *TypeChecker) inferFunctionType(fn *ast.FunctionExpression) ast.Type {

	fnType := &ast.FunctionType{Token: fn.Token, Kind: ast.TypeFunction}

	for _, arg := range fn.Args {
		fnType.Args = append(fnType.Args, arg.Type)
	}

	fnType.Return = append(fnType.Return, fn.Return.RtTypes...)

	return fnType
}

func (tc *TypeChecker) inferFunctionExpr(call *ast.CallExpression) ast.Type {

	sym := tc.symtab.CurrentScope.Resolve(call.Function.Value)

	if sym == nil {
		tc.add(tc.error(call.Token, "function ", call.Function.Value, " wasn't found"))
		return nil
	}

	return sym.Kind
}

func (tc *TypeChecker) checkCallAgainst(fnType ast.Type, call *ast.CallExpression) ast.Type {
	ft, ok := fnType.(*ast.FunctionType)
	if !ok {
		tc.add(tc.error(call.Token, "attempted to call non-function type ", fnType))
		return nil
	}

	if len(call.Args) != len(ft.Args) {
		tc.add(tc.error(call.Token, "function expects  args, got ", len(ft.Args), len(call.Args)))
		return ft.Return[0]
	}
	for i, arg := range call.Args {
		expected := ft.Args[i]
		inferred := tc.inferExpr(arg)

		errMsg := fmt.Sprintf("type mismatch on %v function params, expected %v, instead got %v as type of specified value", tc.highlight(call.Function.Value, Yellow), tc.highlight(expected, Red), tc.highlight(inferred, Green))

		if !tc.typesCompatible(expected, inferred) {
			tc.add(tc.error(arg.GetToken(), errMsg))
		}
	}

	return ft.Return[0] // single return supported for now
}

func (tc *TypeChecker) inferPrimitiveType(expr ast.Expression) ast.Type {

	switch e := expr.(type) {
	case *ast.StringLiteral:
		return &ast.PrimitiveType{
			Token: e.GetToken(),
			Kind:  ast.TypeString,
		}

	case *ast.BooleanLiteral:
		return &ast.PrimitiveType{
			Token: e.GetToken(),
			Kind:  ast.TypeBool,
		}

	case *ast.CharLiteral:
		return &ast.PrimitiveType{
			Token: e.GetToken(),
			Kind:  ast.TypeChar,
		}

	case *ast.NulLiteral:
		return &ast.PrimitiveType{
			Token: e.Token,
			Kind:  ast.TypeAny,
		}

	case *ast.IntegerLiteral:
		// check where does the int literal bounds are
		if e.Value >= 0 {
			switch {
			case e.Value <= math.MaxUint8:
				return &ast.PrimitiveType{
					Token: e.GetToken(),
					Kind:  ast.TypeUInt8,
					Size:  8,
				}
			case e.Value <= math.MaxUint16:
				return &ast.PrimitiveType{
					Token: e.GetToken(),
					Kind:  ast.TypeUInt16,
					Size:  16,
				}
			case e.Value <= math.MaxUint32:
				return &ast.PrimitiveType{
					Token: e.GetToken(),
					Kind:  ast.TypeUInt32,
					Size:  32,
				}

			default:
				return &ast.PrimitiveType{
					Token: e.GetToken(),
					Kind:  ast.TypeUInt64,
					Size:  64,
				}
			}
		} else {

			switch {

			case e.Value <= math.MaxInt8 && e.Value >= math.MinInt8:
				return &ast.PrimitiveType{
					Token:  e.GetToken(),
					Kind:   ast.TypeInt8,
					Size:   8,
					Signed: true,
				}
			case e.Value <= math.MaxInt16 && e.Value >= math.MinInt16:
				return &ast.PrimitiveType{
					Token:  e.GetToken(),
					Kind:   ast.TypeInt16,
					Size:   16,
					Signed: true,
				}

			case e.Value <= math.MaxInt32 && e.Value >= math.MinInt32:
				return &ast.PrimitiveType{
					Token:  e.GetToken(),
					Kind:   ast.TypeInt32,
					Size:   32,
					Signed: true,
				}

			default:
				return &ast.PrimitiveType{
					Token:  e.GetToken(),
					Kind:   ast.TypeInt64,
					Size:   64,
					Signed: true,
				}
			}
		}

	case *ast.FloatLiteral:
		if e.Value <= math.MaxFloat32 {
			return &ast.PrimitiveType{
				Token: e.GetToken(),
				Kind:  ast.TypeFloat32,
				Size:  32,
			}
		}

		if e.Value <= math.MaxFloat64 {
			return &ast.PrimitiveType{
				Token: e.GetToken(),
				Kind:  ast.TypeFloat64,
				Size:  64,
			}
		}

		return nil

	default:
		return nil
	}
}

func (tc *TypeChecker) checkMapType(etp, itp *ast.CompositeType) bool {
	return tc.typesCompatible(etp.LeftType, itp.LeftType) && tc.typesCompatible(etp.RightType, itp.RightType)
}

func (tc *TypeChecker) inferMapType(expr *ast.MapLiteral) ast.Type {
	mapType := expr.Type.(*ast.CompositeType)

	// check the type of each key-value pair
	for key, value := range expr.Pairs {
		inferredKeyType := tc.inferExpr(key)

		if !tc.typesCompatible(mapType.LeftType, inferredKeyType) {
			errMsg := fmt.Sprintf("%v key type isn't compatible with explicit type (%v), consider changing the key's value to the corresponding type", key.String(), inferredKeyType)
			tc.add(tc.error(key.GetToken(), errMsg))
			return nil
		}

		inferredValueType := tc.inferExpr(value)

		if !tc.typesCompatible(mapType.RightType, inferredValueType) {
			errMsg := fmt.Sprintf("%v value type isn't compatible with explicit type (%v), consider changing the key's value to the corresponding type", value.String(), inferredValueType)
			tc.add(tc.error(value.GetToken(), errMsg))
			return nil
		}
	}

	return mapType
}

func (tc *TypeChecker) inferArrayType(expr *ast.ArrayLiteral) ast.Type {
	arrayType := expr.Type.(*ast.CompositeType)

	if arrayType.Size.Value == 0 {
		errMsg := "declaring an array with size 0 isn't allowed, either set the proper size, or make it dynamic"
		tc.add(tc.error(expr.Token, errMsg))
		return nil
	}

	//check the size
	if arrayType.Size.Value > 0 && arrayType.Size.Value < int64(len(expr.Elements)) {
		errMsg := fmt.Sprintf("number of elements > declared size type, see for yourself, declaration states size is %d, number of elements in the array %d", arrayType.Size.Value, len(expr.Elements))
		tc.add(tc.error(expr.Token, errMsg))
		return nil
	}

	// check the type of each element

	for _, element := range expr.Elements {
		inferredElementType := tc.inferExpr(element)

		if !tc.typesCompatible(arrayType.LeftType, inferredElementType) {
			errMsg := fmt.Sprintf("%v element type isn't compatible with explicit type (%v), consider changing the value of the element", element.String(), inferredElementType)
			tc.add(tc.error(element.GetToken(), errMsg))
			return nil
		}
	}

	return arrayType
}

func (tc *TypeChecker) checkArrayType(etp, itp *ast.CompositeType) bool {

	// check against the inferred type (itp)
	if itp.Size.Value == 0 {
		errMsg := "doesn't make sense to declare an array with a fixed size of 0, if u want a fixed size array, size should be > 0, or make it dynamic with [..] syntax"
		tc.add(tc.error(itp.Token, errMsg))
		return false
	}

	if etp.Size.Value == 0 {
		errMsg := "the explicit type contains a fixed size of 0, this isn't allowed, if u want a fixed size array, size should be > 0, or make it dynamic with [..] syntax"
		tc.add(tc.error(etp.Token, errMsg))
		return false
	}

	if itp.Size.Value != etp.Size.Value {
		if itp.Size.Value > etp.Size.Value {
			if etp.Size.Value == -1 {
				errMsg := fmt.Sprintf("explicit type, states that the variable should be a dynamic array, instead we got a fixed size array initialization with %d as it's size, change the explicit type or change the size of initialized array to dynamic with [..] syntax", itp.Size.Value)
				tc.add(tc.error(etp.Token, errMsg))
			} else {
				errMsg := fmt.Sprintf("array was declared with a fixed size of %d, and inferred array size is %d, so either set the correct size or change the fixed size to dynamic using [..] syntax", etp.Size.Value, itp.Size.Value)
				tc.add(tc.error(etp.Token, errMsg))
			}
		} else {
			if itp.Size.Value == -1 {
				errMsg := fmt.Sprintf("explicit type, states that the variable should be an array with a fixed size of %d, instead we got a dynamic array initialization, change the explicit type or change the size of initialized array to fixed size like [%d] syntax", etp.Size.Value, etp.Size.Value)
				tc.add(tc.error(itp.Token, errMsg))
			} else {
				errMsg := fmt.Sprintf("array was declared with a fixed size of %d, and inferred array size is %d, so either set the correct size or change the fixed size to dynamic using [..] syntax", etp.Size.Value, itp.Size.Value)
				tc.add(tc.error(itp.Token, errMsg))
			}
		}
		return false
	}

	return tc.typesCompatible(etp.LeftType, itp.LeftType)
}

func (tc *TypeChecker) typesCompatible(expected, inferred ast.Type) bool {
	if expected == nil || inferred == nil {
		return false
	}

	if expected.Type() != inferred.Type() {
		return false
	}

	// switch based on all types
	switch etp := expected.(type) {
	case *ast.PrimitiveType:
		itp := inferred.(*ast.PrimitiveType)

		if etp.Size < itp.Size {
			return false
		}

		if etp.Kind != ast.TypeFloat32 && etp.Kind != ast.TypeFloat64 {
			if etp.Signed != itp.Signed {
				return false
			}
		}
		return true

	case *ast.CompositeType:
		itp := inferred.(*ast.CompositeType)
		if etp.Kind == ast.TypeMap {
			return tc.checkMapType(etp, itp)
		} else {
			return tc.checkArrayType(etp, itp)
		}

	}

	return false
}
