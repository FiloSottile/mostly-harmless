//go:build js && wasm

package main

import "syscall/js"

func main() {
	js.Global().Set("amendIssuer", js.FuncOf(amendIssuerJS))
	select {}
}

// amendIssuerJS wraps amendIssuer for JavaScript. It takes the issuer and child
// PEM strings and returns an object {pem: string, error: string}; exactly one of
// the two fields is non-empty.
func amendIssuerJS(this js.Value, args []js.Value) (result any) {
	defer func() {
		if r := recover(); r != nil {
			result = map[string]any{"pem": "", "error": "internal error: " + toString(r)}
		}
	}()

	if len(args) != 2 {
		return map[string]any{"pem": "", "error": "expected two arguments: issuer and child PEM"}
	}
	out, err := amendIssuer([]byte(args[0].String()), []byte(args[1].String()))
	if err != nil {
		return map[string]any{"pem": "", "error": err.Error()}
	}
	return map[string]any{"pem": string(out), "error": ""}
}

func toString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	if e, ok := v.(error); ok {
		return e.Error()
	}
	return "unknown"
}
