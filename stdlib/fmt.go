package stdlib

import (
	"blk/object"
	"fmt"
)

var fmtModule = object.Module{
	"print":   &object.BuiltinFn{Fn: print},
	"println": &object.BuiltinFn{Fn: println},
}

func print(args ...object.Object) object.Object {
	printedArgs := prettifyArgs(args...)
	fmt.Print(printedArgs...)
	return nil
}

func println(args ...object.Object) object.Object {
	printedArgs := prettifyArgs(args...)
	fmt.Println(printedArgs...)
	return nil
}

func prettifyArgs(args ...object.Object) []any {
	var printedArgs []any
	for _, arg := range args {
		value := arg.Inspect()
		printedArgs = append(printedArgs, value)
	}
	return printedArgs
}
