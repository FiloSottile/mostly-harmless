//+build gofuzz

package bug

func Fuzz(input []byte) int {
	ParseStrings(input)
	return 1
}
