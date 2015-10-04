package bug

// ParseStrings parses a list of strings encoded in a binary format
// where a single-byte value is followed by a string of that length
// Note: ParseStrings takes unvalidated input from the network
func ParseStrings(input []byte) (result []string) {
	for len(input) > 0 {
		length := input[0]
		input = input[1:]
		result = append(result, string(input[:length]))
		input = input[length:]
	}
	return
}
