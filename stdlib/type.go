package stdlib

import (
	"blk/object"
)

// type module definition
var typeModule = object.Module{
	"INTEGER": &object.String{
		Value: object.INTEGER_OBJ,
	},
	"FLOAT": &object.String{
		Value: object.FLOAT_OBJ,
	},
	"STRING": &object.String{
		Value: object.STRING_OBJ,
	},
	"CHAR": &object.String{
		Value: object.CHAR_OBJ,
	},
	"BOOLEAN": &object.String{
		Value: object.BOOLEAN_OBJ,
	},
}
