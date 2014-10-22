// Package nvram provides an interface to the Mac NVRAM chips on OS X.
//
// It's built upon the heavily stripped C code of the native nvram tool.
//
// The only type supported for the values is data (arbitrary strings in Go)
// both reading and writing. Names must be alphanumeric.
//
// Setting requires superuser privileges.
package nvram

/*
#cgo LDFLAGS: -framework CoreFoundation -framework IOKit

#import <stdlib.h>

unsigned int Setup(char **error);
void Teardown(unsigned int gOptionsRef);
int Get(char *name, char **value, char **error, unsigned int gOptionsRef);
int Set(char *name, char *value, int length, char **error, unsigned int gOptionsRef);
int Delete(char *name, char **error, unsigned int gOptionsRef);
*/
import "C"

import (
	"errors"
	"unsafe"
)

func setup() (C.uint, error) {
	var errStr *C.char
	ref := C.Setup(&errStr)
	if ref == 0 {
		err := errors.New(C.GoString(errStr))
		C.free(unsafe.Pointer(errStr))
		return 0, err
	}
	return ref, nil
}

func teardown(ref C.uint) {
	C.Teardown(ref)
}

// Get retrieves a value stored with a given name from the NVRAM. The value is
// returned as a string of bytes, as stored. An emptry string is returned if a
// value with that name is not found.
func Get(name string) (string, error) {
	ref, err := setup()
	if err != nil {
		return "", err
	}
	defer teardown(ref)

	nameStr := C.CString(name)
	defer C.free(unsafe.Pointer(nameStr))

	var value *C.char
	var errStr *C.char
	length := C.Get(nameStr, &value, &errStr, ref)

	if length == -1 {
		err := errors.New(C.GoString(errStr))
		C.free(unsafe.Pointer(errStr))
		return "", err
	} else if length == -2 {
		return "", nil
	}

	result := C.GoStringN(value, length)

	C.free(unsafe.Pointer(value))

	return result, nil
}

// Set stores a value under the given name. Value can be an arbitrary string,
// name must be alphanumeric.
func Set(name string, value string) error {
	ref, err := setup()
	if err != nil {
		return err
	}
	defer teardown(ref)

	nameStr := C.CString(name)
	defer C.free(unsafe.Pointer(nameStr))
	valueStr := C.CString(value)
	defer C.free(unsafe.Pointer(valueStr))

	var errStr *C.char
	fail := C.Set(nameStr, valueStr, C.int(len(value)), &errStr, ref)

	if fail != 0 {
		err := errors.New(C.GoString(errStr))
		C.free(unsafe.Pointer(errStr))
		return err
	}

	return nil
}

func Delete(name string) error {
	ref, err := setup()
	if err != nil {
		return err
	}
	defer teardown(ref)

	nameStr := C.CString(name)
	defer C.free(unsafe.Pointer(nameStr))

	var errStr *C.char
	fail := C.Delete(nameStr, &errStr, ref)

	if fail != 0 {
		err := errors.New(C.GoString(errStr))
		C.free(unsafe.Pointer(errStr))
		return err
	}

	return nil
}
