package interpreter

import (
	"blk/ast"
	"blk/lexer"
	"blk/object"
	"fmt"
)

var (
	TRUE  = &object.Boolean{Value: true}
	FALSE = &object.Boolean{Value: false}
)

func Eval(node ast.Node) object.Object {

	switch nd := node.(type) {
	case *ast.Program:
		return evalStatements(nd.Statements)

	case *ast.ExpressionStatement:
		return Eval(nd.Expression)

	case *ast.IntegerLiteral:
		return &object.Integer{
			Value: nd.Value,
		}
	case *ast.FloatLiteral:
		return &object.Float{
			Value: nd.Value,
		}

	case *ast.BooleanLiteral:
		return nativeBooleanObject(nd.Value)

	case *ast.UnaryExpression:
		right := Eval(nd.Right)
		return evalUnaryExpression(nd.Operator, right)

	case *ast.BinaryExpression:
		left := Eval(nd.Left)
		right := Eval(nd.Right)
		return evalBinaryExpression(nd.Operator, left, right)

	}

	return nil
}

func evalStatements(stmts []ast.Statement) object.Object {
	var result object.Object
	for _, statement := range stmts {
		result = Eval(statement)
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

func evalUnaryExpression(op string, right object.Object) object.Object {
	switch op {
	case lexer.TokenExclamation:
		// check the right side
		if right.Type() == object.BOOLEAN_OBJ {
			return evalBangOperatorExpression(right.(*object.Boolean))
		}
		// throw an error
		fmt.Println("ERROR: ! operator can only be applied on boolean values")
	case lexer.TokenMinus:
		// support for both ints and floats
		return evalMinusPrefixOperatorExpression(right)
	default:
		return FALSE
	}

	return nil
}

func evalBangOperatorExpression(right *object.Boolean) *object.Boolean {
	if right.Value {
		return FALSE
	} else {
		return TRUE
	}
}

func evalMinusPrefixOperatorExpression(right object.Object) object.Object {
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
		fmt.Println("ERROR: - operator can only be applied on integer or float values")
	}

	return nil
}

func evalBinaryExpression(op string, left, right object.Object) object.Object {
	switch {
	case left.Type() == object.INTEGER_OBJ && right.Type() == object.INTEGER_OBJ:
		return evalIntegerInfixExpression(op, left.(*object.Integer), right.(*object.Integer))

	case left.Type() == object.Float_OBJ && right.Type() == object.Float_OBJ:
		return evalFloatInfixExpression(op, left.(*object.Float), right.(*object.Float))

	case left.Type() == object.BOOLEAN_OBJ || right.Type() == object.BOOLEAN_OBJ:
		// this not allowed at all (no operations on booleans)
		// the only op allowed are (&&, ||)
		return evalBooleanInfixExpression(op, left.(*object.Boolean), right.(*object.Boolean))

	default:
		fmt.Println("NOT SUPPORTED YET", op)
	}

	return nil
}

func evalIntegerInfixExpression(op string, left, right *object.Integer) object.Object {
	switch op {
	// arithmetic operations
	case lexer.TokenMultiply:
		return &object.Integer{
			Value: left.Value * right.Value,
		}
	case lexer.TokenSlash:
		return &object.Integer{
			Value: left.Value / right.Value,
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

	default:
		fmt.Printf("ERROR: %v operator, isn't allowed on integers\n", op)
	}

	return nil
}

func evalFloatInfixExpression(op string, left, right *object.Float) object.Object {
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

	default:
		fmt.Printf("ERROR: %v operator, isn't allowed on floats\n", op)
	}

	return nil
}

func evalBooleanInfixExpression(op string, left, right *object.Boolean) object.Object {
	switch op {
	case lexer.TokenEquals:
		return nativeBooleanObject(left.Value == right.Value)
	case lexer.TokenNotEquals:
		return nativeBooleanObject(left.Value != right.Value)
	case lexer.TokenAnd:
		return nativeBooleanObject(left.Value && right.Value)
	case lexer.TokenOr:
		return nativeBooleanObject(left.Value || right.Value)

	default:
		// error
		fmt.Printf("ERROR: %v operator, isn't allowed on booleans\n", op)
	}

	return nil
}
