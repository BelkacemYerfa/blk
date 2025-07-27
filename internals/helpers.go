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
	if v1.GetType() != v2.GetType() {
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
