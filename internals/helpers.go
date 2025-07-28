package internals

import (
	"blk/parser"
	"fmt"
	"strconv"
)

func ParseToNodeType(nodeType parser.Type) parser.Type {
	// parse the structure and construct the node in NodeType interface
	// ! WTF IS THIS, this works but no, since we're calling this just to parse the type cause it is returned as flat and in NodeTypeFormat
	tokens := parser.NewLexer("", nodeType.String()).Tokenize()
	returnType := parser.NewParser(tokens, "").ParseType()
	return returnType.(parser.Type)
}

func CountChildTypes(nodeType parser.Type) int {
	count := 0

	ndType := nodeType.(*parser.NodeType)

	if ndType.ChildType != nil {
		return CountChildTypes(ndType.ChildType) + 1
	}

	return count
}

// checks equality recursively on v1 and v2
// v1 is the user defined type, v2 is the inferred type
// return a report on where the error happened
func DeepEqualOnNodeType(v1, v2 *parser.NodeType) (bool, *parser.NodeType) {

	if v1 == nil || v2 == nil {
		if v1 == nil {
			return false, v1
		} else {
			return false, v2
		}
	}

	if v1.Type != v2.Type {
		return false, v2
	}

	if len(v1.Size) > 0 {
		fixedSized, _ := strconv.Atoi(v1.Size)
		if len(v2.Size) > 0 {
			v2Size, _ := strconv.Atoi(v2.Size)
			fmt.Println(v1, v2)
			if fixedSized < v2Size {
				return false, v2
			}
		}
	}

	if v1.ChildType == nil && v2.ChildType == nil {
		return true, nil
	}

	if v1.ChildType == nil || v2.ChildType == nil {
		if v1.ChildType == nil {
			return false, v1
		} else {
			return false, v2
		}
	}

	return DeepEqualOnNodeType(v1.ChildType, v2.ChildType)
}

func DeepEqualOnMapType(v1, v2 parser.Type) (bool, parser.Type) {
	if v1.GetType() != v2.GetType() {
		return false, v2
	}

	switch tp1 := v1.(type) {
	case *parser.NodeType:
		switch tp2 := v2.(type) {
		case *parser.NodeType:

			if tp1.ChildType == nil && tp2.ChildType == nil {
				return true, nil
			}

			if tp1.ChildType == nil || tp2.ChildType == nil {
				if tp1.ChildType == nil {
					return false, tp1
				} else {
					return false, tp2
				}
			}

			return DeepEqualOnNodeType(tp1.ChildType, tp2.ChildType)
		case *parser.MapType:
			return false, v2
		}
	case *parser.MapType:
		switch tp2 := v2.(type) {
		case *parser.MapType:
			if tp1.Left == nil && tp2.Left == nil {
				// fallthrough
			} else if tp1.Left == nil || tp2.Left == nil {
				// return the one that has the error
				if tp1.Left == nil {
					return false, v1
				} else {
					return false, v2
				}
			} else {

				leftEqual, leftErr := DeepEqualOnMapType(tp1.Left, tp2.Left)
				if leftErr != nil {
					return false, leftErr
				}
				if !leftEqual {
					return false, nil // or appropriate error
				}
			}

			if tp1.Right == nil && tp2.Right == nil {
				return true, nil
			} else if tp1.Right == nil || tp2.Right == nil {
				if tp1.Right == nil {
					return false, v1
				} else {
					return false, v2
				}
			} else {
				// Both right sides exist, check if they're equal
				return DeepEqualOnMapType(tp1.Right, tp2.Right)
			}
		case *parser.NodeType:
			return false, v2
		}
	}

	return false, nil
}
