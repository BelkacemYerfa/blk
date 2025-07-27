package internals

import (
	"blk/parser"
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

func CheckEqualityOnFieldSize(v *parser.NodeType, size string) bool {
	if v.Size != size && v.Type == "array" {
		return false
	}
	if v.ChildType == nil {
		return true
	}

	return CheckEqualityOnFieldSize(v.ChildType, size)
}
