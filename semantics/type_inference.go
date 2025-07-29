package semantics

import (
	"blk/internals"
	"blk/parser"
	"fmt"
	"slices"
	"strconv"
	"strings"
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

func (tar *typeAliasResolver) normalizeType(nodeType parser.Expression) parser.Type {
	switch tp := nodeType.(type) {
	case *parser.NodeType:
		if tp.ChildType != nil {
			return &parser.NodeType{
				Token:     tp.Token,
				Type:      tp.Type,
				ChildType: tar.normalizeType(tp.ChildType).(*parser.NodeType),
				Size:      tp.Size,
			}
		}
		// follow of the bug is here, since it didn't find a child it called this directly
		return &parser.NodeType{
			Token: tp.Token,
			Type:  tar.resolveAlias(tp.Type),
			Size:  tp.Size,
		}
	case *parser.MapType:
		res := &parser.MapType{
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
		typeName = tar.normalizeType(alias.DeclNode.(*parser.TypeStatement).Value.(parser.Type)).String()
	}
	return typeName
}

type TypeInference struct {
	CurrSymbol    *SymbolInfo
	Collector     *internals.ErrorCollector
	aliasResolver *typeAliasResolver
	symbols       *symbolResolver
}

func NewTypeInference(errCollector *internals.ErrorCollector, aliasRs *typeAliasResolver, symbolRs *symbolResolver) *TypeInference {
	return &TypeInference{
		Collector:     errCollector,
		aliasResolver: aliasRs,
		symbols:       symbolRs,
	}
}

func (tie *TypeInference) insertUniqueErrors(errMsg error) {
	_, found := slices.BinarySearchFunc(tie.Collector.Errors, errMsg, func(a, b error) int {
		return strings.Compare(a.Error(), b.Error())
	})
	if !found {
		tie.Collector.Add(errMsg)
	}
}

func (ti *TypeInference) visitIdentifier(expr *parser.Identifier) *SymbolInfo {
	// if (identifier) check if it declared or not
	ident, isMatched := ti.symbols.Resolve(expr.Value)

	if !isMatched {
		errMsg := ("ERROR: identifier, needs to be declared before it gets used")
		ti.Collector.Add(ti.Collector.Error(expr.Token, errMsg))
	}

	return ident
}

func (ti *TypeInference) inferAssociatedValueType(expr parser.Expression) parser.Type {

	switch ep := expr.(type) {
	// atomic types
	case *parser.StringLiteral:
		return &parser.NodeType{
			Type: parser.StringType,
		}
	case *parser.BooleanLiteral:
		return &parser.NodeType{
			Type: parser.BoolType,
		}
	case *parser.IntegerLiteral:
		return &parser.NodeType{
			Type: parser.IntType,
		}
	case *parser.FloatLiteral:
		return &parser.NodeType{
			Type: parser.FloatType,
		}

	case *parser.ArrayLiteral:
		return ti.inferArrayType(ep)
	case *parser.MapLiteral:
		return ti.inferMapType(ep)
	case *parser.Identifier:
		return ti.inferIdentifierType(ep)
	case *parser.StructInstanceExpression:
		return ti.inferStructInstanceType(ep)
	case *parser.IndexExpression:
		return ti.inferIndexAccessType(ep)
	case *parser.CallExpression:
		return ti.inferCallExpressionType(ep)
	case *parser.UnaryExpression:
		return ti.inferUnaryExpressionType(ep)
	case *parser.BinaryExpression:
		return ti.inferBinaryExpressionType(ep)
	}

	return &parser.NodeType{}
}

func (ti *TypeInference) inferArrayType(expr *parser.ArrayLiteral) parser.Type {

	if len(expr.Elements) == 0 {
		return ti.CurrSymbol.Type.(*parser.NodeType)
	}

	firstElem := &parser.NodeType{}

	for idx, elem := range expr.Elements {
		resType := ti.inferAssociatedValueType(elem)
		resType = ti.aliasResolver.normalizeType(resType)
		if idx == 0 {
			firstElem = resType.(*parser.NodeType)
		}
		if firstElem.Type != resType.(*parser.NodeType).Type {
			errMsg := fmt.Sprintf("ERROR: multitude of different types in the array (%v,%v,...etc), remove incompatible types", firstElem, resType)
			expr.Token.Text = expr.String()
			ti.insertUniqueErrors(ti.Collector.Error(expr.Token, errMsg))
		}
	}

	// read the type

	return &parser.NodeType{
		Token:     expr.Token,
		Type:      "array",
		ChildType: firstElem,
		Size:      fmt.Sprint(len(expr.Elements)),
	}

}

func (ti *TypeInference) inferMapType(expr *parser.MapLiteral) parser.Type {
	if len(expr.Pairs) == 0 {
		return ti.CurrSymbol.Type.(*parser.NodeType)
	}
	// use interface for readability (preferred over any)
	var keyElem interface{}
	var valElem interface{}
	idx := 0
	for key, value := range expr.Pairs {
		// key part
		resType := ti.inferAssociatedValueType(key)
		resType = ti.aliasResolver.normalizeType(resType)
		if idx == 0 {
			keyElem = resType
		}
		switch rst := keyElem.(type) {
		case *parser.NodeType:
			if rst.Type != resType.GetType() {
				errMsg := fmt.Sprintf("ERROR: multitude of different types in the array (%v,%v,...etc), remove incompatible types", keyElem, resType)
				ti.insertUniqueErrors(ti.Collector.Error(key.GetToken(), errMsg))
			}
		case *parser.MapType:
			if rst.Type != resType.GetType() {
				errMsg := fmt.Sprintf("ERROR: multitude of different types in the array (%v,%v,...etc), remove incompatible types", keyElem, resType)
				ti.insertUniqueErrors(ti.Collector.Error(key.GetToken(), errMsg))
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
		case *parser.NodeType:
			if rst.Type != resType.GetType() {
				errMsg := fmt.Sprintf("ERROR: multitude of different types in the array (%v,%v,...etc), remove incompatible types", keyElem, resType)
				ti.insertUniqueErrors(ti.Collector.Error(value.GetToken(), errMsg))
			}
		case *parser.MapType:
			if rst.Type != resType.GetType() {
				errMsg := fmt.Sprintf("ERROR: multitude of different types in the array (%v,%v,...etc), remove incompatible types", keyElem, resType)
				ti.insertUniqueErrors(ti.Collector.Error(value.GetToken(), errMsg))
			}
		default:
		}
	}

	return &parser.MapType{
		Token: expr.Token,
		Type:  "map",
		Left:  keyElem.(parser.Type),
		Right: valElem.(parser.Type),
	}
}

func (ti *TypeInference) inferIdentifierType(expr *parser.Identifier) parser.Type {
	// call the visitIdentifier
	sym := ti.visitIdentifier(expr)

	if sym == nil {
		return nil
	}

	switch node := sym.DeclNode.(type) {
	case *parser.LetStatement:
		return ti.inferAssociatedValueType(node.Value)
	case *parser.StructInstanceExpression:
		return ti.inferAssociatedValueType(node.Left)
	case *parser.ArgExpression:
		// this for args type of function definitions
		return node.Type.(parser.Type)
	default:
	}

	return &parser.NodeType{
		Token: expr.Token,
		Type:  expr.Value,
	}
}

func (ti *TypeInference) inferStructInstanceType(expr *parser.StructInstanceExpression) parser.Type {
	// check if the types are compatible with the definition
	// rule the left is only an identifier, if it is something else add an error to the collector and return
	switch lf := expr.Left.(type) {
	case *parser.Identifier:
		// fall through
	default:
		errMsg := fmt.Sprintf("ERROR: (%v) type can't be used here, only identifiers", lf)
		// TODO: enhance the token position placement
		tok := expr.Left.GetToken()
		ti.Collector.Add(ti.Collector.Error(tok, errMsg))
		return nil
	}

	sym := ti.visitIdentifier(expr.Left.(*parser.Identifier))
	structDef := &parser.StructStatement{}

	if sym == nil {
		return nil
	}

	switch structDf := sym.DeclNode.(type) {
	case *parser.StructStatement:
		structDef = structDf
	case *parser.TypeStatement:
		structDef = ti.visitIdentifier(
			&parser.Identifier{
				Value: structDf.Value.(*parser.NodeType).Type,
			},
		).DeclNode.(*parser.StructStatement)
	default:
	}

	var keyElem interface{}
	idx := 0
	for id, elem := range expr.Body {
		// check the types are evaluated correctly on the fields
		resType := structDef.Body[id]
		if idx == 0 {
			keyElem = resType
			idx++
		}
		switch rst := keyElem.(type) {
		case *parser.NodeType:
			if rst.Type != resType.Value.String() {
				errMsg := fmt.Sprintf("ERROR: multitude of different types in the array (%v,%v,...etc), remove incompatible types", keyElem, resType)
				ti.insertUniqueErrors(ti.Collector.Error(elem.Value.GetToken(), errMsg))
			}
		case *parser.MapType:
			if rst.Type != resType.Value.String() {
				errMsg := fmt.Sprintf("ERROR: multitude of different types in the array (%v,%v,...etc), remove incompatible types", keyElem, resType)
				ti.insertUniqueErrors(ti.Collector.Error(elem.Value.GetToken(), errMsg))
			}
		default:
		}
	}

	return ti.aliasResolver.normalizeType(sym.Type)
}

func (ti *TypeInference) inferIndexAccessType(expr *parser.IndexExpression) parser.Type {
	// check what the left side is an int if it is an array
	// also if it is a map allow indexing with key name that correspond to that type

	resType := ti.inferAssociatedValueType(expr.Left)
	switch rst := resType.(type) {
	case *parser.NodeType:
		// bug here cause of that type infer with call expression and indexing
		if len(rst.Size) > 0 {
			// only for fixed size arrays
			fixedSized, _ := strconv.Atoi(rst.Size)
			index, _ := strconv.Atoi(expr.Index.String())

			if index > fixedSized-1 {
				errMsg := fmt.Sprintf("ERROR: index out of bound, array size %d", fixedSized)
				expr.Token.Text = expr.String()
				ti.Collector.Add(ti.Collector.Error(expr.Token, errMsg))
			}
		}
		// break at this point, when using the array map
		indexType := ti.inferAssociatedValueType(expr.Index)
		if indexType.String() != "int" && rst.Type == "array" {
			errMsg := fmt.Sprintf("ERROR: can't use %v as index, index should be of type int %v", indexType, rst)
			expr.Token.Text = expr.String()
			ti.Collector.Add(ti.Collector.Error(expr.Token, errMsg))
			return nil
		}
		// problem is here
		// ti.Collector.Add(rst.Type) // prints array(int) instead of array
		if rst.ChildType != nil {
			return rst.ChildType
		} else {
			// parse the structure and construct the node in NodeType interface
			return internals.ParseToNodeType(rst).(*parser.NodeType).ChildType
		}
	case *parser.MapType:
		// get the type of the current side
		// This returns all the actual type of the left side, need the type based on the nest level
		inferType := ti.inferAssociatedValueType(expr.Left)
		tempType := inferType
		switch tp := inferType.(type) {
		case *parser.NodeType:
			for tp.ChildType != nil {
				tempType = tp.ChildType
			}
		case *parser.MapType:
			for tp.Left != nil {
				tempType = tp.Left
				if leftType, ok := tp.Left.(*parser.MapType); ok {
					tp = leftType
				} else {
					break
				}
			}

			for tp.Right != nil {
				inferType = tp.Right
				if RightType, ok := tp.Right.(*parser.MapType); ok {
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
			ti.Collector.Add(ti.Collector.Error(expr.Token, errMsg))
			return nil
		}

		return inferType
	default:
	}

	return nil
}

func (ti *TypeInference) inferCallExpressionType(expr *parser.CallExpression) parser.Type {
	sym := ti.visitIdentifier(&expr.Function)

	if sym == nil {
		return nil
	}

	// calls may return hashMaps, check for that also
	return internals.ParseToNodeType(ti.aliasResolver.normalizeType(sym.Type))
}

func (ti *TypeInference) inferUnaryExpressionType(expr *parser.UnaryExpression) parser.Type {
	if ti.CurrSymbol.Type.(*parser.NodeType).Type == parser.BoolType && expr.Operator == "-" {
		errMsg := "ERROR: can't use operator (-) with boolean types, only operator (!) is allowed"
		ti.insertUniqueErrors(ti.Collector.Error(expr.Token, errMsg))
	}

	return ti.inferAssociatedValueType(expr.Right)
}

func (ti *TypeInference) inferBinaryExpressionType(expr *parser.BinaryExpression) parser.Type {
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
		ti.Collector.Add(ti.Collector.Error(expr.Token, errMsg))
		return nil
	}

	operator := expr.Operator

	switch operator {
	case "==", "!=":
		return &parser.NodeType{
			Token: expr.Token,
			Type:  parser.BoolType,
		}
	case "+":
		if leftType.String() != parser.StringType && leftType.String() != parser.FloatType && leftType.String() != parser.IntType {
			errMsg := fmt.Sprintf(
				"ERROR: (%s) isn't allowed on (%v) type", operator, leftType.String(),
			)
			ti.Collector.Add(ti.Collector.Error(expr.Token, errMsg))
			return nil
		} else {
			return &parser.NodeType{
				Token: expr.Token,
				Type:  leftType.String(),
			}
		}
	case "-", "/", "*", "%", ">=", "<=", ">", "<":
		switch leftType.String() {
		case parser.IntType:
			// skip
		case parser.FloatType:
			// skip
		default:
			// throw the error here
			errMsg := fmt.Sprintf(
				"ERROR: (%s) isn't allowed on (%v) type", operator, leftType.String(),
			)
			ti.Collector.Add(ti.Collector.Error(expr.Token, errMsg))
			return nil
		}
		return &parser.NodeType{
			Token: expr.Token,
			Type:  leftType.String(),
		}
	case "&&", "||":
		if leftType.String() != parser.BoolType {
			errMsg := fmt.Sprintf(
				"ERROR: (%s) isn't allowed on %s type", operator, leftType.String(),
			)
			ti.insertUniqueErrors(ti.Collector.Error(expr.Token, errMsg))
		}
	default:
		// this will return the first part if nothing there matched
		return &parser.NodeType{
			Token: expr.Token,
			Type:  leftType.String(),
		}
	}

	return &parser.NodeType{
		Token: expr.Token,
		Type:  parser.BoolType,
	}
}
