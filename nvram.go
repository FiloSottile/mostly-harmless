package main

/*
#cgo LDFLAGS: -framework CoreFoundation -framework IOKit

#import <stdlib.h>

int Setup(char **error);
void Teardown(void);
int Get(char *name, char **value, char **error);
int Set(char *name, char *value, int length, char **error);
int Delete(char *name, char **error);
*/
import "C"

import (
	"errors"
	"log"
	"unsafe"
)

func main() {
	err := Setup()
	if err != nil {
		log.Fatalln(err.Error())
	}
	defer Teardown()

	res, err := Get("filippo")
	if err != nil {
		log.Fatalln(err.Error())
	}

	log.Printf("% x\n", res)

	err = Set("filippo", "42è\x00\xff")
	if err != nil {
		log.Fatalln(err.Error())
	}

	log.Println("Set done")

	res, err = Get("filippo")
	if err != nil {
		log.Fatalln(err.Error())
	}

	log.Printf("% x\n", res)

	err = Delete("filippo")
	if err != nil {
		log.Fatalln(err.Error())
	}

	log.Println("Delete done")

	res, err = Get("filippo")
	if err != nil {
		log.Fatalln(err.Error())
	}

	log.Printf("% x\n", res)
}

func Setup() error {
	var errStr *C.char
	fail := C.Setup(&errStr)
	if fail != 0 {
		err := errors.New(C.GoString(errStr))
		C.free(unsafe.Pointer(errStr))
		return err
	}
	return nil
}

func Teardown() {
	C.Teardown()
}

func Get(name string) (string, error) {
	nameStr := C.CString(name)
	defer C.free(unsafe.Pointer(nameStr))

	var value *C.char
	var errStr *C.char
	length := C.Get(nameStr, &value, &errStr)

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

func Set(name string, value string) error {
	nameStr := C.CString(name)
	defer C.free(unsafe.Pointer(nameStr))
	valueStr := C.CString(value)
	defer C.free(unsafe.Pointer(valueStr))

	var errStr *C.char
	fail := C.Set(nameStr, valueStr, C.int(len(value)), &errStr)

	if fail != 0 {
		err := errors.New(C.GoString(errStr))
		C.free(unsafe.Pointer(errStr))
		return err
	}

	return nil
}

func Delete(name string) error {
	nameStr := C.CString(name)
	defer C.free(unsafe.Pointer(nameStr))

	var errStr *C.char
	fail := C.Delete(nameStr, &errStr)

	if fail != 0 {
		err := errors.New(C.GoString(errStr))
		C.free(unsafe.Pointer(errStr))
		return err
	}

	return nil
}
