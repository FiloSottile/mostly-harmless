package main

/*
#cgo CFLAGS: -Wall
#cgo LDFLAGS: -framework CoreFoundation -framework IOKit
void setup(void);
void teardown(void);
void PrintOFVariables(void);
*/
import "C"

func main() {
	C.setup()
	C.PrintOFVariables()
	C.teardown()
}
