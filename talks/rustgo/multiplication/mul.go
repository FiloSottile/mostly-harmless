//go:binary-only-package

package multiplication

import _ "unsafe"

//go:cgo_import_static multiply_two
//go:cgo_import_dynamic multiply_two
//go:linkname multiply_two multiply_two
var multiply_two uintptr
var _multiply_two = &multiply_two

func MultiplyByTwo(n uint64) uint64
