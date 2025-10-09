package semantic

import (
	"blk/ast"
	"blk/lexer"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Typechecker implementation
// Responsible of providing an implementation of blk type checker that will be using bidirectional typechecking approach to verify & analyze the types

type TypeChecker struct {
	filePath string
	symtab   *SymbolTable
	errors   *ErrorCollector
}

func NewTypeChecker(filepath string, symtab *SymbolTable, errors *ErrorCollector) *TypeChecker {
	return &TypeChecker{
		filePath: filepath,
		symtab:   symtab,
		errors:   errors,
	}
}

func (tc *TypeChecker) check(program *ast.Program) {
	for _, stmt := range program.Statements {
		tc.checkStmt(stmt)
	}
}

func (tc *TypeChecker) checkStmt(stmt ast.Statement) {
	if tc.symtab.GlobalScope == tc.symtab.CurrentScope {
		// only declaration and imports allowed
		switch stmt.(type) {
		case *ast.Declaration, *ast.ImportStatement, *ast.TypeDeclaration:
		default:
			(tc.errors.error(ERROR, stmt.GetToken(), "the global scope only allows for declaration or import statements, everything else if forbidden"))
			return
		}
	}

	if tc.symtab.CurrentScope != tc.symtab.GlobalScope {
		if _, ok := stmt.(*ast.ImportStatement); ok {
			(tc.errors.error(ERROR, stmt.GetToken(), "import statements are only allowed in the global scope, for organization purposes keep them at the top of the file"))
			return
		}
	}

	switch s := stmt.(type) {
	case *ast.Declaration:
		tc.checkDeclaration(s)

	case *ast.ScopeStatement:
		tc.inferBlockExprType(s.Body)

	}
}

func (tc *TypeChecker) checkDeclaration(d *ast.Declaration) {
	isExplicit := d.Type != nil
	isInit := d.Value != nil

	name := d.Name[0].String()

	nit, ok := d.Directive[lexer.TokenNoInit]

	if ok && !isExplicit {
		tc.errors.error(ERROR, nit.Token, name, " requires the type to be explicit since it uses the #no_init directive, either consider giving the type or an init value removing the #no_init directive")
		return
	}

	if ok && isInit {
		tc.errors.error(ERROR, nit.Token, name, " is declared with #no_init directive, meaning it shouldn't have an expression as a value, but got ", d.Value, " consider removing it")
		return
	}

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
		(tc.errors.error(ERROR, d.Token, "variable with ", name, " wasn't found"))
		return
	}

	if isExplicit && isInit {
		// check the explicit type against the inferred type
		inferred := tc.inferExpr(expr)

		if !tc.typesCompatible(d.Type, inferred) {
			errMsg := fmt.Sprintf("type mismatch, explicit type %v doesn't match inferred type (%v), change the explicit type, or let the compiler infer it with := syntax", tc.errors.highlight(d.Type, Red), tc.errors.highlight(inferred, Green))
			(tc.errors.error(ERROR, expr.GetToken(), errMsg))
			return
		}
	}

	if !isExplicit && isInit {
		// no explicit type, type will get inferred from the assign expression
		d.Type = tc.inferExpr(expr)
	}
}

func (tc *TypeChecker) unalias(tp ast.Type) ast.Type {

	alsTp, ok := tp.(*ast.AliasType)
	if !ok {
		return tp
	}

	// search for the alias
	sym := tc.symtab.CurrentScope.Resolve(alsTp.Alias.Value)

	if sym == nil {
		(tc.errors.error(ERROR, alsTp.Token, "type ", alsTp.Alias, " wasn't found"))
		return nil
	}

	sym.Used = true

	if sym.Kind.Type() == ast.TypeAlias {
		return tc.unalias(sym.Kind)
	}

	return sym.Kind
}

func (tc *TypeChecker) inferExpr(expr ast.Expression) ast.Type {
	switch e := expr.(type) {

	case *ast.UnaryExpression:
		return tc.inferUnaryExprType(e)

	case *ast.BinaryExpression:
		return tc.inferBinaryExprType(e)

	case *ast.BlockExpression:
		return tc.inferBlockExprType(e)

	case *ast.IfExpression:
		return tc.inferIfExprType(e)

	case *ast.SwitchExpression:
		return tc.inferSwitchExprType(e)

	case *ast.FunctionExpression:
		return tc.inferFunctionType(e)

	case *ast.Identifier:
		return tc.inferIdentifierType(e)

	case *ast.CallExpression:
		fnType := tc.inferFunctionExpr(e)
		fnSign := tc.functionSignature(fnType, e)
		return tc.checkCallAgainst(fnType, fnSign, e)

	case *ast.IntegerLiteral, *ast.StringLiteral, *ast.CharLiteral, *ast.NulLiteral, *ast.FloatLiteral, *ast.BooleanLiteral:
		return tc.inferPrimitiveType(e)

	case *ast.MapLiteral:
		return tc.inferMapType(e)

	case *ast.ArrayLiteral:
		return tc.inferArrayType(e)

	case *ast.IndexExpression:
		return tc.inferIndexType(e)

	case *ast.CastExpression:
		// check if expression is supported first
		switch e.TargetExpression.(type) {
		case *ast.Identifier, ast.Literal, *ast.CallExpression, *ast.CastExpression, *ast.UnaryExpression:
			exprType := tc.inferExpr(e.TargetExpression)
			return tc.checkTypeAgainst(exprType, e)

		default:
			errMsg := fmt.Sprintf("%v expression isn't supported with cast expression, the only supported ones are literal (floats, ints, ...), identifiers, functions call and nested cast expression", tc.errors.highlight(e.TargetExpression.String(), Red))
			(tc.errors.error(ERROR, e.Token, errMsg))
			return nil
		}

	default:
		(tc.errors.error(ERROR, expr.GetToken(), "cannot infer type for expression"))
		return nil
	}

}

func (tc *TypeChecker) inferUnaryExprType(expr *ast.UnaryExpression) ast.Type {
	rightType := tc.inferExpr(expr.Right)

	switch expr.Operator {
	case "!":
		if rightType.Type() != ast.TypeBool {
			(tc.errors.error(ERROR, expr.Right.GetToken(), fmt.Errorf("bang operator can only be used with boolean, but the received type is %s", rightType)))
			return nil
		}

		return rightType

	case "~":
		if rightType.Type() > ast.TypeUInt64 {
			(tc.errors.error(ERROR, expr.Right.GetToken(), fmt.Errorf("bitwise not operator ~ can only be applied on signed int or unsigned int, but the received type is %v", rightType)))
			return nil
		}

		return rightType

	case "-":
		if rightType.Type() > ast.TypeFloat64 {
			(tc.errors.error(ERROR, expr.Right.GetToken(), fmt.Errorf("- operator can only be applied either on signed int or unsigned int or floats, but the received type is %v", rightType)))
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

	if leftType == nil || rightType == nil {
		return nil
	}

	if !tc.typesCompatible(leftType, rightType) {
		(tc.errors.error(ERROR, expr.Token, "sides of binary expression are not equals, left ", tc.errors.highlight(leftType, Yellow), " right ", tc.errors.highlight(rightType, Green)))
		return nil
	}

	switch expr.Operator {

	case "+", "-", "*", "/", "%":
		// cases allowed: string (plus only) , int, uint, floats

		if leftType.Type() > ast.TypeString || rightType.Type() > ast.TypeString {
			(tc.errors.error(ERROR, expr.Token, "arithmetic operators can only be applied on (integers, unsigned integers, floats, strings), but the received type is ", rightType))
			return nil
		}

		switch leftType.Type() {
		case ast.TypeString:
			if expr.Operator != "+" {
				(tc.errors.error(ERROR, expr.Token, "the only arithmetic operation allowed on strings is +, but received ", expr.Operator, " as operator"))
				return nil
			}

			return leftType

		case ast.TypeFloat32, ast.TypeFloat64:
			if expr.Operator == "%" {
				(tc.errors.error(ERROR, expr.Token, "% (modules) operator isn't allowed on float, it is only allowed on ints"))
				return nil
			}

			return leftType

		default:
			return leftType

		}

	case "==", "!=":
		// cases allowed: bool, string, int, uint, floats

		if leftType.Type() > ast.TypeBool || rightType.Type() > ast.TypeBool {
			(tc.errors.error(ERROR, expr.Token, "arithmetic operators can only be applied on (integers, unsigned integers, floats, strings), but the received type is ", rightType))
			return nil
		}

		return &ast.PrimitiveType{Token: expr.Token, Kind: ast.TypeBool}

	case "<", ">", "<=", ">=":
		// cases allowed: int, uint, floats

		if leftType.Type() > ast.TypeBool || rightType.Type() > ast.TypeBool {
			(tc.errors.error(ERROR, expr.Token, "comparison operators can only be applied on (integers, unsigned integers, floats, strings), but the received type is ", rightType))
			return nil
		}

		return &ast.PrimitiveType{Token: expr.Token, Kind: ast.TypeBool}

	case "||", "&&":
		//cases allowed: booleans

		if leftType.Type() != ast.TypeBool || rightType.Type() != ast.TypeBool {
			(tc.errors.error(ERROR, expr.Token, "arithmetic operators can only be applied on (integers, unsigned integers, floats, strings), but the received type is ", rightType))
			return nil
		}

		return &ast.PrimitiveType{Token: expr.Token, Kind: ast.TypeBool}

	case "<<", ">>", "^", "&", "|":
		// cases allowed int, uint
		if leftType.Type() > ast.TypeUInt64 || rightType.Type() > ast.TypeUInt64 {
			(tc.errors.error(ERROR, expr.Token, "bitwise operators can only be applied on (integers, unsigned integers), but the received type is ", rightType))
			return nil
		}

		return leftType

	}

	return leftType
}

func (tc *TypeChecker) checkFunctionBody(fn *ast.FunctionExpression) {
	// check the type against the last statement of the block
	fnType := tc.inferFunctionType(fn).(*ast.FunctionType)
	blockType := tc.inferBlockExprType(fn.Body)

	if fnType == nil || blockType == nil {
		return
	}

	if !tc.typesCompatible(fnType.Return[0], blockType) {
		errMsg := ""
		if fnType.Return[0].Type() == ast.TypeVoid {
			errMsg = fmt.Sprintf("type mismatch where function doesn't expect a return, but got %v as return type", tc.errors.highlight(blockType, Red))
		} else {
			errMsg = fmt.Sprintf("type mismatch between the returned typed %v, and the expected return type %v", tc.errors.highlight(blockType, Red), tc.errors.highlight(fnType.Return[0], Yellow))
		}
		(tc.errors.error(ERROR, fn.Token, errMsg))
	}
}

func (tc *TypeChecker) inferBlockExprType(block *ast.BlockExpression) ast.Type {
	// check if the last statement is expr stmt, if it is, the block will be of it's type
	// a unique case is with return where the type of that block needs to be of the same type of the return
	// for the last statement the return is optional, in that case this will refer to first point made about last expr stmt

	if len(block.Body) == 0 {
		return &ast.PrimitiveType{
			Kind: ast.TypeVoid,
		}
	}

	for _, stmt := range block.Body {
		tc.checkStmt(stmt)

		if rtStmt, ok := stmt.(*ast.ReturnStatement); ok {
			if len(rtStmt.ReturnValues) == 0 {
				return &ast.PrimitiveType{
					Kind: ast.TypeVoid,
				}
			}

			// TODO: currently only one is supported for simplicity, refactor later to support multi return values
			return tc.inferExpr(rtStmt.ReturnValues[0])
		}
	}

	lastStmt := block.Body[len(block.Body)-1]

	if exprStmt, ok := lastStmt.(*ast.ExpressionStatement); ok {
		return tc.inferExpr(exprStmt.Expression)
	}

	return &ast.PrimitiveType{
		Kind: ast.TypeVoid,
	}
}

func (tc *TypeChecker) inferIfExprType(ifExpr *ast.IfExpression) ast.Type {

	conditionType := tc.inferExpr(ifExpr.Condition)

	if conditionType == nil {
		return nil
	}

	// check that the condition evaluates to a boolean
	if conditionType.Type() != ast.TypeBool {
		errMsg := fmt.Sprintf("condition on if statement needs to be of type bool, instead got type %v", tc.errors.highlight(conditionType, Red))
		(tc.errors.error(ERROR, ifExpr.Condition.GetToken(), errMsg))
		return nil
	}

	// TODO: think about how the checks will be done in here

	tt := tc.inferBlockExprType(ifExpr.Consequence)

	if ifExpr.Alternative != nil {
		return tc.inferExpr(ifExpr.Alternative)
	}

	return tt
}

func (tc *TypeChecker) inferSwitchExprType(switchExpr *ast.SwitchExpression) ast.Type {
	switchTargetType := tc.inferExpr(switchExpr.Condition)

	if switchTargetType == nil {
		return nil
	}

	tc.symtab.EnterScope()
	defer tc.symtab.ExitScope()

	var bodyType ast.Type

	for _, cs := range switchExpr.Cases {
		// check the arm pattern of each case
		for _, csArm := range cs.ArmPattern {
			csArmType := tc.inferExpr(csArm)

			if csArmType == nil {
				return nil
			}

			if csArmType.Type() != switchTargetType.Type() && csArmType.Type() != ast.TypeAny {
				errMsg := fmt.Sprintf("%v arm is of type %v that doesn't match %v type, consider changing the value type", csArm, tc.errors.highlight(csArmType, Red), tc.errors.highlight(switchTargetType, Yellow))
				(tc.errors.error(ERROR, switchExpr.Condition.GetToken(), errMsg))
			}
		}

		// check the body of each case
		bodyType = tc.inferBlockExprType(cs.Body)
	}

	return bodyType
}

func (tc *TypeChecker) inferIndexType(idxExpr *ast.IndexExpression) ast.Type {
	leftType := tc.inferExpr(idxExpr.Left)

	if leftType == nil {
		return nil
	}

	if leftType.Type() != ast.TypeArray && leftType.Type() != ast.TypeMap {
		errMsg := fmt.Sprintf("left side of index expression should be either an array or map, instead got %v", tc.errors.highlight(leftType, Red))
		(tc.errors.error(ERROR, idxExpr.Left.GetToken(), errMsg))
		return nil
	}

	castLeftType := leftType.(*ast.CompositeType)
	startType := tc.inferExpr(idxExpr.Start)

	if leftType.Type() == ast.TypeMap {
		keyType := castLeftType.LeftType

		if !tc.typesCompatible(keyType, startType) {
			errMsg := fmt.Sprintf("provided key type is %v, doesn't match the original expected type %v", tc.errors.highlight(startType, Red), tc.errors.highlight(keyType, Yellow))
			(tc.errors.error(ERROR, idxExpr.Left.GetToken(), errMsg))
			return nil
		}

		// check if end exists
		if idxExpr.Range {
			(tc.errors.error(ERROR, idxExpr.Left.GetToken(), "left side of index expression is of type map, thus u can't use range with map"))
			return nil
		}
	}

	if leftType.Type() == ast.TypeArray {

		if startType.Type() < ast.TypeSInt8 || startType.Type() > ast.TypeUInt64 {
			errMsg := fmt.Sprintf("left side of index expression is of type array, thus the start index should be of type int, instead got %v", tc.errors.highlight(leftType, Red))
			(tc.errors.error(ERROR, idxExpr.Left.GetToken(), errMsg))
			return nil
		}

		if idxExpr.Range {
			endType := tc.inferExpr(idxExpr.End)

			if endType != nil && (startType.Type() < ast.TypeSInt8 || startType.Type() > ast.TypeUInt64) {
				errMsg := fmt.Sprintf("left side of index expression is of type array, thus the end bound should be of type int, instead got %v", tc.errors.highlight(leftType, Red))
				(tc.errors.error(ERROR, idxExpr.Left.GetToken(), errMsg))
				return nil
			}
		}
	}

	if leftType.Type() == ast.TypeMap {
		return castLeftType.RightType
	}

	return castLeftType.LeftType
}

func (tc *TypeChecker) inferFunctionType(fn *ast.FunctionExpression) ast.Type {

	fnType := &ast.FunctionType{Token: fn.Token, Kind: ast.TypeFunction}

	for _, arg := range fn.Args {
		fnType.Args = append(fnType.Args, arg.Type)
	}

	if len(fn.Return.RtTypes) == 0 {
		fnType.Return = append(fnType.Return, &ast.PrimitiveType{
			Kind: ast.TypeVoid,
		})
	} else {
		fnType.Return = append(fnType.Return, fn.Return.RtTypes...)
	}

	return fnType
}

func (tc *TypeChecker) functionSignature(fnType ast.Type, call *ast.CallExpression) *ast.FunctionSignature {

	fn := tc.symtab.CurrentScope.Resolve(call.Function.Value)
	if fn == nil {
		(tc.errors.error(ERROR, call.Token, "function ", call.Function.Value, " wasn't found"))
		return nil
	}

	ft, ok := fn.Kind.(*ast.FunctionType)
	if !ok {
		(tc.errors.error(ERROR, call.Token, "attempted to call non-function type ", fnType))
		return nil
	}

	// this cast is required to get the signature
	fnDecl, ok := fn.DeclNode.(*ast.Declaration).Value[0].(*ast.FunctionExpression)

	if !ok {
		(tc.errors.error(ERROR, call.Token, "associated value ", call.Function.Value, " is not a functions"))
		return nil
	}

	fnSignature := &ast.FunctionSignature{Token: ft.Token, Kind: ast.TypeFunction, Args: map[string]*ast.Arg{}}

	for _, arg := range fnDecl.Args {
		fnSignature.Args[arg.Name.Value] = &ast.Arg{
			Token: arg.Token,
			Name:  arg.Name,
			Type:  arg.Type,
		}
	}

	if len(ft.Return) == 0 {
		fnSignature.Return = append(fnSignature.Return, &ast.PrimitiveType{
			Kind: ast.TypeVoid,
		})
	} else {
		fnSignature.Return = append(fnSignature.Return, ft.Return...)
	}

	return fnSignature
}

func (tc *TypeChecker) inferIdentifierType(ident *ast.Identifier) ast.Type {
	sym := tc.symtab.CurrentScope.Resolve(ident.Value)

	if sym == nil {
		(tc.errors.error(ERROR, ident.Token, "identifier ", ident.Value, " wasn't found"))
		return nil
	}

	sym.Used = true
	return sym.Kind
}

func (tc *TypeChecker) inferFunctionExpr(call *ast.CallExpression) ast.Type {

	sym := tc.symtab.CurrentScope.Resolve(call.Function.Value)
	if sym == nil {
		(tc.errors.error(ERROR, call.Token, "function ", call.Function.Value, " wasn't found"))
		return nil
	}

	return sym.Kind
}

func (tc *TypeChecker) checkCallAgainst(fnType ast.Type, fnSign *ast.FunctionSignature, call *ast.CallExpression) ast.Type {
	// get the function Signature
	ft, ok := fnType.(*ast.FunctionType)
	if !ok {
		(tc.errors.error(ERROR, call.Token, "attempted to call non-function type ", fnType))
		return nil
	}

	if fnSign == nil {
		(tc.errors.error(ERROR, call.Token, "function signature issues ", fnSign))
		return nil
	}

	if len(call.Args) > len(ft.Args) {
		(tc.errors.error(ERROR, call.Token, "function expects ", len(ft.Args), " instead got ", len(call.Args)))
		return ft.Return[0]
	}

	for i, arg := range call.Args {
		expected := ft.Args[i]
		inferred := tc.inferExpr(arg.Value)

		if arg.Name != nil {
			exp, found := fnSign.Args[arg.Name.Value]

			if !found {
				errMsg := fmt.Sprintf("argument %v wasn't found in the function", tc.errors.highlight(arg.Name, Yellow))
				(tc.errors.error(ERROR, arg.Token, errMsg))
				return nil
			}

			expected = exp.Type
		}

		if !tc.typesCompatible(expected, inferred) {
			errMsg := fmt.Sprintf("type mismatch on %v function params, expected %v, instead got %v as type of specified value", tc.errors.highlight(call.Function.Value, Yellow), tc.errors.highlight(expected, Red), tc.errors.highlight(inferred, Green))
			(tc.errors.error(ERROR, arg.Token, errMsg))
		}
	}

	return ft.Return[0] // single return supported for now
}

func (tc *TypeChecker) inferPrimitiveType(expr ast.Expression) ast.Type {

	switch e := expr.(type) {
	case *ast.StringLiteral:
		if strings.HasPrefix(e.Value, "0x") || strings.HasPrefix(e.Value, "0o") || strings.HasPrefix(e.Value, "0b") {
			// parse the other representation
			base := 2
			switch {
			case strings.HasPrefix(e.Value, "0x"):
				base = 16
			case strings.HasPrefix(e.Value, "0o"):
				base = 8
			}

			tokenValue := e.Value
			e.Value = e.Value[2:]
			_, err := strconv.ParseInt(e.Value, base, 64)

			if err != nil {
				if errors.Is(err, strconv.ErrRange) {
					errMsg := fmt.Sprintf("%v is out of range, max values with int is %v", tc.errors.highlight(tokenValue, Yellow), tc.errors.highlight(math.MaxInt64, Yellow))
					tc.errors.error(ERROR, e.Token, errMsg)
				} else {
					tc.errors.error(ERROR, e.Token, err)
				}
				return nil
			}

			return &ast.PrimitiveType{
				Token: e.GetToken(),
				Kind:  ast.TypeUInt64, // switch later to sint type
			}
		}

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
					Kind:   ast.TypeSInt8,
					Size:   8,
					Signed: true,
				}
			case e.Value <= math.MaxInt16 && e.Value >= math.MinInt16:
				return &ast.PrimitiveType{
					Token:  e.GetToken(),
					Kind:   ast.TypeSInt16,
					Size:   16,
					Signed: true,
				}

			case e.Value <= math.MaxInt32 && e.Value >= math.MinInt32:
				return &ast.PrimitiveType{
					Token:  e.GetToken(),
					Kind:   ast.TypeSInt32,
					Size:   32,
					Signed: true,
				}

			default:
				return &ast.PrimitiveType{
					Token:  e.GetToken(),
					Kind:   ast.TypeSInt64,
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
			(tc.errors.error(ERROR, key.GetToken(), errMsg))
			return nil
		}

		inferredValueType := tc.inferExpr(value)

		if !tc.typesCompatible(mapType.RightType, inferredValueType) {
			errMsg := fmt.Sprintf("%v value type isn't compatible with explicit type (%v), consider changing the key's value to the corresponding type", value.String(), inferredValueType)
			(tc.errors.error(ERROR, value.GetToken(), errMsg))
			return nil
		}
	}

	return mapType
}

func (tc *TypeChecker) inferArrayType(expr *ast.ArrayLiteral) ast.Type {
	arrayType := expr.Type.(*ast.CompositeType)

	if arrayType.Size.Value == 0 {
		errMsg := "declaring an array with size 0 isn't allowed, either set the proper size, or make it dynamic"
		(tc.errors.error(ERROR, expr.Token, errMsg))
		return nil
	}

	//check the size
	if arrayType.Size.Value > 0 && arrayType.Size.Value < int64(len(expr.Elements)) {
		errMsg := fmt.Sprintf("number of elements > declared size type, see for yourself, declaration states size is %d, number of elements in the array %d", arrayType.Size.Value, len(expr.Elements))
		(tc.errors.error(ERROR, expr.Token, errMsg))
		return nil
	}

	// check the type of each element

	for _, element := range expr.Elements {
		inferredElementType := tc.inferExpr(element)

		if !tc.typesCompatible(arrayType.LeftType, inferredElementType) {
			errMsg := fmt.Sprintf("%v element type isn't compatible with explicit type (%v), consider changing the value of the element", element.String(), inferredElementType)
			(tc.errors.error(ERROR, element.GetToken(), errMsg))
			return nil
		}
	}

	return arrayType
}

func (tc *TypeChecker) checkArrayType(etp, itp *ast.CompositeType) bool {

	// check against the inferred type (itp)
	if itp.Size.Value == 0 {
		errMsg := "doesn't make sense to declare an array with a fixed size of 0, if u want a fixed size array, size should be > 0, or make it dynamic with [..] syntax"
		(tc.errors.error(ERROR, itp.Token, errMsg))
		return false
	}

	if etp.Size.Value == 0 {
		errMsg := "the explicit type contains a fixed size of 0, this isn't allowed, if u want a fixed size array, size should be > 0, or make it dynamic with [..] syntax"
		(tc.errors.error(ERROR, etp.Token, errMsg))
		return false
	}

	if itp.Size.Value != etp.Size.Value {
		if itp.Size.Value > etp.Size.Value {
			if etp.Size.Value == -1 {
				errMsg := fmt.Sprintf("explicit type, states that the variable should be a dynamic array, instead we got a fixed size array initialization with %d as it's size, change the explicit type or change the size of initialized array to dynamic with [..] syntax", itp.Size.Value)
				(tc.errors.error(ERROR, etp.Token, errMsg))
			} else {
				errMsg := fmt.Sprintf("array was declared with a fixed size of %d, and inferred array size is %d, so either set the correct size or change the fixed size to dynamic using [..] syntax", etp.Size.Value, itp.Size.Value)
				(tc.errors.error(ERROR, etp.Token, errMsg))
			}
		} else {
			if itp.Size.Value == -1 {
				errMsg := fmt.Sprintf("explicit type, states that the variable should be an array with a fixed size of %d, instead we got a dynamic array initialization, change the explicit type or change the size of initialized array to fixed size like [%d] syntax", etp.Size.Value, etp.Size.Value)
				(tc.errors.error(ERROR, itp.Token, errMsg))
			} else {
				errMsg := fmt.Sprintf("array was declared with a fixed size of %d, and inferred array size is %d, so either set the correct size or change the fixed size to dynamic using [..] syntax", etp.Size.Value, itp.Size.Value)
				(tc.errors.error(ERROR, itp.Token, errMsg))
			}
		}
		return false
	}

	return tc.typesCompatible(etp.LeftType, itp.LeftType)
}

func (tc *TypeChecker) checkPrimitiveTypeCastAbility(ctt, exprPrimitive *ast.PrimitiveType) (*ast.PrimitiveType, error) {

	switch {
	case ctt.Kind == exprPrimitive.Kind:
		// Same type
		return exprPrimitive, nil

	case ctt.Kind >= ast.TypeSInt8 && ctt.Kind <= ast.TypeFloat64:
		// Casting TO numeric
		switch {
		case exprPrimitive.Kind >= ast.TypeSInt8 && exprPrimitive.Kind <= ast.TypeFloat64:
			// Numeric -> Numeric
			if exprPrimitive.Kind > ctt.Kind {
				errMsg := fmt.Sprintf("Be careful, casting type %v into type %v will result in some information loss", tc.errors.highlight(exprPrimitive, Red), tc.errors.highlight(ctt, Yellow))
				return exprPrimitive, tc.errors.error(ERROR, exprPrimitive.Token, errMsg)
			}
			return ctt, nil

		case exprPrimitive.Kind == ast.TypeBool:
			// Bool -> Numeric (true=1, false=0)
			return ctt, nil

		case exprPrimitive.Kind == ast.TypeChar:
			// Char -> Numeric ('A' -> 65)
			return ctt, nil

		case exprPrimitive.Kind == ast.TypeString:
			errMsg := fmt.Sprintf("converting %v into %v, isn't allowed", tc.errors.highlight(exprPrimitive, Red), tc.errors.highlight(ctt, Yellow))
			return nil, tc.errors.error(ERROR, exprPrimitive.Token, errMsg)
		}

	case ctt.Kind == ast.TypeChar:
		switch exprPrimitive.Kind {

		case ast.TypeSInt8, ast.TypeSInt16, ast.TypeSInt32, ast.TypeSInt64,
			ast.TypeUInt8, ast.TypeUInt16, ast.TypeUInt32, ast.TypeUInt64:
			// Numeric -> Char (runtime check: must fit in range)
			return ctt, nil
		case ast.TypeString:
			// String -> Char (runtime check: must be len==1)
			return ctt, nil

		default:
			errMsg := fmt.Sprintf("converting %v into %v, isn't allowed", tc.errors.highlight(exprPrimitive, Red), tc.errors.highlight(ctt, Yellow))
			return nil, tc.errors.error(ERROR, exprPrimitive.Token, errMsg)
		}

	case ctt.Kind == ast.TypeBool:
		switch exprPrimitive.Kind {

		case ast.TypeSInt8, ast.TypeSInt16, ast.TypeSInt32, ast.TypeSInt64,
			ast.TypeUInt8, ast.TypeUInt16, ast.TypeUInt32, ast.TypeUInt64,
			ast.TypeFloat32, ast.TypeFloat64, ast.TypeChar:
			// Numeric/Char -> Bool (0=false, else true)
			return ctt, nil

		default:
			errMsg := fmt.Sprintf("converting %v into %v, isn't allowed", tc.errors.highlight(exprPrimitive, Red), tc.errors.highlight(ctt, Yellow))
			return nil, tc.errors.error(ERROR, exprPrimitive.Token, errMsg)
		}

	case ctt.Kind == ast.TypeString:
		// Everything -> String
		return ctt, nil

	default:
		errMsg := fmt.Sprintf("converting %v into %v, isn't allowed", tc.errors.highlight(exprPrimitive, Red), tc.errors.highlight(ctt, Yellow))
		return nil, tc.errors.error(ERROR, exprPrimitive.Token, errMsg)
	}

	// unreachable
	return nil, nil
}

func (tc *TypeChecker) checkTypeAgainst(exprType ast.Type, cast *ast.CastExpression) ast.Type {

	// force := cast.Directives.Kind == ast.ForceDirective
	targetExpr := cast.TargetExpression
	castToType := cast.TargetType

	switch ctt := castToType.(type) {
	case *ast.PrimitiveType:
		// cast to the same type
		exprPrimitive, ok := exprType.(*ast.PrimitiveType)
		if !ok {
			errMsg := fmt.Sprintf("types need to be of the same category constructor, meaning if the provided type is a primitive, the expr needs to give back a primitive, if it is a composite type, it needs to give back composite type, but we received %v as cast type, and expr gives back %v type", tc.errors.highlight(castToType, Green), tc.errors.highlight(exprType, Red))
			(tc.errors.error(ERROR, targetExpr.GetToken(), errMsg))
			return nil
		}

		res, err := tc.checkPrimitiveTypeCastAbility(ctt, exprPrimitive)
		if err != nil {
			tc.errors.error(ERROR, cast.Token, err)
		}
		return res

	case *ast.CompositeType:

		exprPrimitive, ok := exprType.(*ast.CompositeType)
		if !ok {
			errMsg := fmt.Sprintf("types need to be of the same category constructor, meaning if the provided type is a primitive, the expr needs to give back a primitive, if it is a composite type, it needs to give back composite type, but we received %v as cast type, and expr gives back %v type", tc.errors.highlight(castToType, Green), tc.errors.highlight(exprType, Red))
			(tc.errors.error(ERROR, targetExpr.GetToken(), errMsg))
			return nil
		}

		if ctt.Kind != exprPrimitive.Kind {
			errMsg := fmt.Sprintf("trying to cast an expression of type %v into %v type isn't allowed", tc.errors.highlight(exprType, Red), tc.errors.highlight(castToType, Green))
			(tc.errors.error(ERROR, targetExpr.GetToken(), errMsg))
			return nil
		}

		if ctt.Kind == ast.TypeArray {
			if ctt.Size.Value != exprPrimitive.Size.Value {
				errMsg := fmt.Sprintf("array size are not equal, expression array size is %v and cast type size is %v, both of them need to be equal", tc.errors.highlight(exprPrimitive.Size, Red), tc.errors.highlight(ctt.Size, Green))
				(tc.errors.error(ERROR, targetExpr.GetToken(), errMsg))
				return nil
			}

			if !tc.typesCompatible(ctt.LeftType, exprPrimitive.LeftType) {
				errMsg := fmt.Sprintf("trying to cast %v into %v isn't allowed", tc.errors.highlight(exprPrimitive, Red), tc.errors.highlight(ctt, Green))
				(tc.errors.error(ERROR, targetExpr.GetToken(), errMsg))

				return nil
			}

			return exprPrimitive

		}

		if ctt.Kind == ast.TypeMap {
			if tp := tc.checkTypeAgainst(ctt.LeftType, &ast.CastExpression{
				TargetType:       exprPrimitive.LeftType,
				TargetExpression: targetExpr,
			}); tp == nil {
				errMsg := fmt.Sprintf("trying to cast %v into %v isn't allowed", tc.errors.highlight(exprPrimitive.LeftType, Red), tc.errors.highlight(ctt.LeftType, Green))
				(tc.errors.error(ERROR, targetExpr.GetToken(), errMsg))

				return nil
			}

			if tp := tc.checkTypeAgainst(ctt.RightType, &ast.CastExpression{
				TargetType:       exprPrimitive.RightType,
				TargetExpression: targetExpr,
			}); tp == nil {
				errMsg := fmt.Sprintf("trying to cast %v into %v isn't allowed", tc.errors.highlight(exprPrimitive.RightType, Red), tc.errors.highlight(ctt.RightType, Green))
				(tc.errors.error(ERROR, targetExpr.GetToken(), errMsg))

				return nil
			}

			return exprPrimitive
		}

	default:
		errMsg := fmt.Sprintf("using %v as casting type isn't supported yet",
			tc.errors.highlight(cast.String(), Red))
		(tc.errors.error(ERROR, cast.Token, errMsg))
		return nil
	}

	return nil
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
