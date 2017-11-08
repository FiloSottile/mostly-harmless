package bug

import "testing"
import "reflect"

func TestModeLen(t *testing.T) {
	result := ParseStrings([]byte("\x0cHello World!"))
	expected := []string{"Hello World!"}
	if !reflect.DeepEqual(result, expected) {
		t.Fatal(result)
	}
}

func TestMultiple(t *testing.T) {
	result := ParseStrings([]byte("\x03Foo\x0cHello World!"))
	expected := []string{"Foo", "Hello World!"}
	if !reflect.DeepEqual(result, expected) {
		t.Fatal(result)
	}
}

func TestEmptyString(t *testing.T) {
	result := ParseStrings([]byte{})
	if len(result) != 0 {
		t.Fatal(result)
	}
}
