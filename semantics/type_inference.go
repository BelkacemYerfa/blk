package semantics

import (
	"blk/ast"
	"blk/internals"
	"fmt"
	"strconv"
)

type typeAliasResolver struct {
	resolver *symbolResolver
	visiting map[string]bool
}

func NewTypeAliasResolver(resolver *symbolResolver) *typeAliasResolver {
	return &typeAliasResolver{
		resolver: resolver,
		visiting: make(map[string]bool),
	}
}

func (tar *typeAliasResolver) normalizeType(nodeType ast.Expression) ast.Type {
	switch tp := nodeType.(type) {
	case *ast.NodeType:
		if tp.ChildType != nil {
			return &ast.NodeType{
				Token:     tp.Token,
				Type:      tp.Type,
				ChildType: tar.normalizeType(tp.ChildType).(*ast.NodeType),
				Size:      tp.Size,
			}
		}
		// follow of the bug is here, since it didn't find a child it called this directly
		return &ast.NodeType{
			Token: tp.Token,
			Type:  tar.resolveAlias(tp.Type),
			Size:  tp.Size,
		}
	case *ast.MapType:
		res := &ast.MapType{
			Token: tp.Token,
			Type:  "map",
		}
		if tp.Left != nil {
			res.Left = tar.normalizeType(tp.Left)
		}

		if tp.Right != nil {
			res.Right = tar.normalizeType(tp.Right)
		}

		return res
	}
	return nil
}

func (tar *typeAliasResolver) resolveAlias(typeName string) string {
	visited := map[string]bool{}
	for {
		if visited[typeName] {
			break // avoid cycles
		}
		visited[typeName] = true

		alias, ok := tar.resolver.Resolve(typeName)
		if !ok {
			return typeName
		}

		if alias.Kind != SymbolType {
			return typeName
		}

		// get the value of the type alias
		// typeName = tar.normalizeType(alias.DeclNode.(*ast.TypeStatement).Value.(ast.Type)).String()
		typeName = alias.Name
	}
	return typeName
}

type TypeInference struct {
	currSymbol    *SymbolInfo
	collector     *internals.ErrorCollector
	aliasResolver *typeAliasResolver
	symbols       *symbolResolver
}

func NewTypeInference(errCollector *internals.ErrorCollector, aliasRs *typeAliasResolver, symbolRs *symbolResolver) *TypeInference {
	return &TypeInference{
		collector:     errCollector,
		aliasResolver: aliasRs,
		symbols:       symbolRs,
	}
}

func (ti *TypeInference) visitIdentifier(expr *ast.Identifier) *SymbolInfo {
	// if (identifier) check if it declared or not
	ident, isMatched := ti.symbols.Resolve(expr.Value)

	if !isMatched {
		errMsg := ("ERROR: identifier, needs to be declared before it gets used")
		ti.collector.Add(ti.collector.Error(expr.Token, errMsg))
	}

	return ident
}

func (ti *TypeInference) inferAssociatedValueType(expr ast.Expression) ast.Type {

	switch ep := expr.(type) {
	// atomic types
	case *ast.StringLiteral:
		return &ast.NodeType{
			Type: ast.StringType,
		}
	case *ast.BooleanLiteral:
		return &ast.NodeType{
			Type: ast.BoolType,
		}
	case *ast.IntegerLiteral:
		return &ast.NodeType{
			Type: ast.IntType,
		}
	case *ast.FloatLiteral:
		return &ast.NodeType{
			Type: ast.FloatType,
		}

	case *ast.ArrayLiteral:
		return ti.inferArrayType(ep)
	case *ast.MapLiteral:
		return ti.inferMapType(ep)
	case *ast.Identifier:
		return ti.inferIdentifierType(ep)
	case *ast.StructInstanceExpression:
		return ti.inferStructInstanceType(ep)
	case *ast.IndexExpression:
		return ti.inferIndexAccessType(ep)
	case *ast.CallExpression:
		return ti.inferCallExpressionType(ep)
	case *ast.UnaryExpression:
		return ti.inferUnaryExpressionType(ep)
	case *ast.BinaryExpression:
		return ti.inferBinaryExpressionType(ep)
	case *ast.MemberShipExpression:
		return ti.inferMembershipExpressionType(ep)
	}

	return &ast.NodeType{}
}

func (ti *TypeInference) inferArrayType(expr *ast.ArrayLiteral) ast.Type {

	if len(expr.Elements) == 0 {
		return ti.currSymbol.Type.(*ast.NodeType)
	}

	firstElem := &ast.NodeType{}

	for idx, elem := range expr.Elements {
		resType := ti.inferAssociatedValueType(elem)
		resType = ti.aliasResolver.normalizeType(resType)
		if idx == 0 {
			firstElem = resType.(*ast.NodeType)
		}
		if firstElem.Type != resType.(*ast.NodeType).Type {
			errMsg := fmt.Sprintf("ERROR: multitude of different types in the array (%v,%v,...etc), remove incompatible types", firstElem, resType)
			expr.Token.Text = expr.String()
			ti.collector.Add(ti.collector.Error(expr.Token, errMsg))
		}
	}

	// read the type

	return &ast.NodeType{
		Token:     expr.Token,
		Type:      "array",
		ChildType: firstElem,
		Size:      fmt.Sprint(len(expr.Elements)),
	}

}

func (ti *TypeInference) inferMapType(expr *ast.MapLiteral) ast.Type {
	if len(expr.Pairs) == 0 {
		return ti.currSymbol.Type.(*ast.NodeType)
	}
	// use interface for readability (preferred over any)
	var keyElem, valElem ast.Type

	idx := 0
	for key, value := range expr.Pairs {
		// key part
		resType := ti.inferAssociatedValueType(key)
		resType = ti.aliasResolver.normalizeType(resType)
		if idx == 0 {
			keyElem = resType
		}
		switch rst := keyElem.(type) {
		case *ast.NodeType:
			if rst.Type != resType.GetType() {
				errMsg := fmt.Sprintf("ERROR: multitude of different types in the array (%v,%v,...etc), remove incompatible types", keyElem, resType)
				ti.collector.Add(ti.collector.Error(key.GetToken(), errMsg))
			}
		case *ast.MapType:
			if rst.Type != resType.GetType() {
				errMsg := fmt.Sprintf("ERROR: multitude of different types in the array (%v,%v,...etc), remove incompatible types", keyElem, resType)
				ti.collector.Add(ti.collector.Error(key.GetToken(), errMsg))
			}
		default:
		}

		// value part
		resType = ti.inferAssociatedValueType(value)
		resType = ti.aliasResolver.normalizeType(resType)
		if idx == 0 {
			valElem = resType
			idx++
		}
		switch rst := valElem.(type) {
		case *ast.NodeType:
			if rst.Type != resType.GetType() {
				errMsg := fmt.Sprintf("ERROR: multitude of different types in the array (%v,%v,...etc), remove incompatible types", keyElem, resType)
				ti.collector.Add(ti.collector.Error(value.GetToken(), errMsg))
			}
		case *ast.MapType:
			if rst.Type != resType.GetType() {
				errMsg := fmt.Sprintf("ERROR: multitude of different types in the array (%v,%v,...etc), remove incompatible types", keyElem, resType)
				ti.collector.Add(ti.collector.Error(value.GetToken(), errMsg))
			}
		default:
		}
	}

	return &ast.MapType{
		Token: expr.Token,
		Type:  "map",
		Left:  keyElem,
		Right: valElem,
	}
}

func (ti *TypeInference) inferIdentifierType(expr *ast.Identifier) ast.Type {
	// here before check if the value is a field in a struct first
	switch cast := ti.currSymbol.DeclNode.(type) {
	case *ast.VarDeclaration:
		structDef := cast.Value.(*ast.StructExpression)
		for idx := range structDef.Body {
			field := structDef.Body[idx]

			if field.Key.Value == expr.Value {
				// cast the key to the function definition
				if typeDef, ok := field.Value.(ast.Type); ok {
					fmt.Println(typeDef)
					return internals.ParseToNodeType(ti.aliasResolver.normalizeType(typeDef))
				}
			}
		}
	case *ast.FunctionExpression:
	// 	return ti.inferAssociatedValueType(cast)
	default:
		// call the visitIdentifier
		sym := ti.visitIdentifier(expr)

		if sym == nil {
			return nil
		}

		switch node := sym.DeclNode.(type) {
		case *ast.VarDeclaration:
			return ti.inferAssociatedValueType(node.Value)
		case *ast.StructInstanceExpression:
			return ti.inferAssociatedValueType(node.Left)
		case *ast.ArgExpression:
			// this for args type of function definitions
			return node.Type.(ast.Type)
		default:
		}
	}

	return &ast.NodeType{
		Token: expr.Token,
		Type:  expr.Value,
	}
}

func (ti *TypeInference) inferStructInstanceType(expr *ast.StructInstanceExpression) ast.Type {
	// check if the types are compatible with the definition
	// rule the left is only an identifier, if it is something else add an error to the collector and return
	switch lf := expr.Left.(type) {
	case *ast.Identifier:
		// fall through
	default:
		errMsg := fmt.Sprintf("ERROR: (%v) type can't be used here, only identifiers", lf)
		// TODO: enhance the token position placement
		tok := expr.Left.GetToken()
		ti.collector.Add(ti.collector.Error(tok, errMsg))
		return nil
	}

	sym := ti.visitIdentifier(expr.Left.(*ast.Identifier))
	structDef := &ast.StructExpression{}

	if sym == nil {
		return nil
	}

	switch structDf := sym.DeclNode.(*ast.VarDeclaration).Value.(type) {
	case *ast.StructExpression:
		structDef = structDf
	// case *ast.TypeStatement:
	// 	structDef = ti.visitIdentifier(
	// 		&ast.Identifier{
	// 			Value: structDf.Value.(*ast.NodeType).Type,
	// 		},
	// 	).DeclNode.(*ast.StructStatement)
	default:
	}

	for id, elem := range expr.Body {
		// check the types are evaluated correctly on the fields
		resType := structDef.Body[id].Value

		// inferred type of the associated value
		inferredType := ti.inferAssociatedValueType(elem.Value)

		// compare the values
		if resType.String() != inferredType.String() {
			errMsg := fmt.Sprintf("ERROR: incompatible types, expected %v, but inferred %v, consider changing the associated type or the assigned value", resType, inferredType)
			ti.collector.Add(ti.collector.Error(elem.Value.GetToken(), errMsg))
		}
	}

	return ti.aliasResolver.normalizeType(sym.Type)
}

func (ti *TypeInference) inferIndexAccessType(expr *ast.IndexExpression) ast.Type {
	// check what the left side is an int if it is an array
	// also if it is a map allow indexing with key name that correspond to that type

	resType := ti.inferAssociatedValueType(expr.Left)
	switch rst := resType.(type) {
	case *ast.NodeType:
		// bug here cause of that type infer with call expression and indexing
		if len(rst.Size) > 0 {
			// only for fixed size arrays
			fixedSized, _ := strconv.Atoi(rst.Size)
			index, _ := strconv.Atoi(expr.Index.String())

			if index > fixedSized-1 {
				errMsg := fmt.Sprintf("ERROR: index out of bound, array size %d", fixedSized)
				expr.Token.Text = expr.String()
				ti.collector.Add(ti.collector.Error(expr.Token, errMsg))
			}
		}
		// break at this point, when using the array map
		indexType := ti.inferAssociatedValueType(expr.Index)
		if indexType.String() != "int" && rst.Type == "array" {
			errMsg := fmt.Sprintf("ERROR: can't use %v as index, index should be of type int %v", indexType, rst)
			expr.Token.Text = expr.String()
			ti.collector.Add(ti.collector.Error(expr.Token, errMsg))
			return nil
		}
		// problem is here
		// ti.collector.Add(rst.Type) // prints array(int) instead of array
		if rst.ChildType != nil {
			return rst.ChildType
		} else {
			// parse the structure and construct the node in NodeType interface
			return internals.ParseToNodeType(rst).(*ast.NodeType).ChildType
		}
	case *ast.MapType:
		// get the type of the current side
		// This returns all the actual type of the left side, need the type based on the nest level
		inferType := ti.inferAssociatedValueType(expr.Left)
		tempType := inferType
		switch tp := inferType.(type) {
		case *ast.NodeType:
			for tp.ChildType != nil {
				tempType = tp.ChildType
			}
		case *ast.MapType:
			for tp.Left != nil {
				tempType = tp.Left
				if leftType, ok := tp.Left.(*ast.MapType); ok {
					tp = leftType
				} else {
					break
				}
			}

			for tp.Right != nil {
				inferType = tp.Right
				if RightType, ok := tp.Right.(*ast.MapType); ok {
					tp = RightType
				} else {
					break
				}
			}
		}

		indexType := ti.inferAssociatedValueType(expr.Index)
		if indexType.String() != tempType.String() {
			errMsg := fmt.Sprintf("ERROR: can't use type (%v) as key in a map, key should be of same type as one defined in map (%s)", indexType, tempType)
			expr.Token.Text = expr.String()
			ti.collector.Add(ti.collector.Error(expr.Token, errMsg))
			return nil
		}

		return inferType
	default:
	}

	return nil
}

func (ti *TypeInference) inferCallExpressionType(expr *ast.CallExpression) ast.Type {
	// check if this function is method or is a function
	sym := ti.currSymbol
	switch cast := sym.DeclNode.(type) {
	case *ast.VarDeclaration:
		structDef := cast.Value.(*ast.StructExpression)
		for idx := range structDef.Body {
			field := structDef.Body[idx]

			if field.Key.Value == expr.Function.Value {
				// cast the key to the function definition
				if functionDef, ok := field.Value.(*ast.FunctionExpression); ok {
					return internals.ParseToNodeType(ti.aliasResolver.normalizeType(functionDef.ReturnType))
				}
			}
		}
	// case *ast.FunctionExpression:

	default:
		sym = ti.visitIdentifier(&expr.Function)

		if sym == nil {
			return nil
		}
	}

	// calls may return hashMaps, check for that also
	return internals.ParseToNodeType(ti.aliasResolver.normalizeType(sym.Type))
}

func (ti *TypeInference) inferUnaryExpressionType(expr *ast.UnaryExpression) ast.Type {
	if ti.currSymbol.Type.(*ast.NodeType).Type == ast.BoolType && expr.Operator == "-" {
		errMsg := "ERROR: can't use operator (-) with boolean types, only operator (!) is allowed"
		ti.collector.Add(ti.collector.Error(expr.Token, errMsg))
	}

	return ti.inferAssociatedValueType(expr.Right)
}

func (ti *TypeInference) inferBinaryExpressionType(expr *ast.BinaryExpression) ast.Type {
	// check if the operation is allowed on that type
	// rule: equality on all
	// comparison only on floats, and ints
	// rule: allow only comparison of the same types
	leftType := ti.inferAssociatedValueType(expr.Left)
	rightType := ti.inferAssociatedValueType(expr.Right)
	if leftType.String() != rightType.String() {
		errMsg := fmt.Sprintf(
			"ERROR: type mismatch, we can't compare 2 different types in a binary expression, left (%v), right (%v)", leftType, rightType,
		)
		expr.Token.Col++
		ti.collector.Add(ti.collector.Error(expr.Token, errMsg))
		return nil
	}

	operator := expr.Operator

	switch operator {
	case "==", "!=":
		return &ast.NodeType{
			Token: expr.Token,
			Type:  ast.BoolType,
		}
	case "+":
		if leftType.String() != ast.StringType && leftType.String() != ast.FloatType && leftType.String() != ast.IntType {
			errMsg := fmt.Sprintf(
				"ERROR: (%s) isn't allowed on (%v) type", operator, leftType.String(),
			)
			ti.collector.Add(ti.collector.Error(expr.Token, errMsg))
			return nil
		} else {
			return &ast.NodeType{
				Token: expr.Token,
				Type:  leftType.String(),
			}
		}
	case "-", "/", "*", "%", ">=", "<=", ">", "<":
		switch leftType.String() {
		case ast.IntType:
			// skip
		case ast.FloatType:
			// skip
		default:
			// throw the error here
			errMsg := fmt.Sprintf(
				"ERROR: (%s) isn't allowed on (%v) type", operator, leftType.String(),
			)
			ti.collector.Add(ti.collector.Error(expr.Token, errMsg))
			return nil
		}
		return &ast.NodeType{
			Token: expr.Token,
			Type:  leftType.String(),
		}
	case "&&", "||":
		if leftType.String() != ast.BoolType {
			errMsg := fmt.Sprintf(
				"ERROR: (%s) isn't allowed on %s type", operator, leftType.String(),
			)
			ti.collector.Add(ti.collector.Error(expr.Token, errMsg))
		}
	default:
		// this will return the first part if nothing there matched
		return &ast.NodeType{
			Token: expr.Token,
			Type:  leftType.String(),
		}
	}

	return &ast.NodeType{
		Token: expr.Token,
		Type:  ast.BoolType,
	}
}

func (ti *TypeInference) inferMembershipExpressionType(expr *ast.MemberShipExpression) ast.Type {
	start, ok := expr.Object.(*ast.Identifier)

	if !ok {
		errMsg := "ERROR: blk language doesn't support bind besides on struct definitions"
		ti.collector.Add(ti.collector.Error(expr.Object.GetToken(), errMsg))
		return nil
	}

	sym, found := ti.symbols.Resolve(start.Value)

	if !found {
		errMsg := fmt.Sprintf(
			"ERROR: (%s) needs to be declare an initialized first", expr.Object)
		ti.collector.Add(ti.collector.Error(expr.Token, errMsg))
		return nil
	}

	// check if it is a struct instance
	instance, isStructInstance := sym.Type.(*ast.StructInstanceExpression)

	if !isStructInstance {
		errMsg := fmt.Sprintf(
			"ERROR: (%s) needs to be of a valid struct instance", expr.Object)
		ti.collector.Add(ti.collector.Error(expr.Token, errMsg))
		return nil
	}

	// check if the property type is built into that struct definition
	ident := ti.visitIdentifier(instance.Left.(*ast.Identifier))

	if ident == nil {
		return nil
	}

	structDef := ident.DeclNode.(*ast.VarDeclaration).Value.(*ast.StructExpression)

	found = false

	for idx := range structDef.Body {
		field := structDef.Body[idx]

		if field.Key.Value == expr.Property.GetToken().Text {
			// check if the property is an identifier field
			if _, ok := expr.Property.(*ast.Identifier); ok {
				found = true
				break
			}

			// checks on the call expression
			if _, ok := field.Value.(*ast.FunctionExpression); ok {
				// check if the property is of type function expression
				if _, ok := expr.Property.(*ast.CallExpression); ok {
					found = true
					break
				}
			}
		}
	}

	if !found {
		errMsg := fmt.Sprintf(
			"ERROR: (%s) needs to exist on the struct definition", expr.Property)
		ti.collector.Add(ti.collector.Error(expr.Property.GetToken(), errMsg))
		return nil
	}

	ti.currSymbol = ident

	return ti.inferAssociatedValueType(expr.Property)
}
