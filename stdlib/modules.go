package stdlib

import "blk/object"

// every module added to the std lib needs to be defined here with a name
var BuiltinModules = map[string]object.Module{
	"math": mathModule,
}
