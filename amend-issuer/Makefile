GOROOT := $(shell go env GOROOT)

.PHONY: build
build: amend.wasm.gz wasm_exec.js

amend.wasm: amend.go wasm.go go.mod
	GOOS=js GOARCH=wasm go build -ldflags="-s -w" -o amend.wasm .

# amend.wasm.gz is committed and decompressed in the browser; amend.wasm is an
# intermediate (see .gitignore).
amend.wasm.gz: amend.wasm
	gzip -9 -c amend.wasm > amend.wasm.gz

wasm_exec.js: $(GOROOT)/lib/wasm/wasm_exec.js
	cp "$(GOROOT)/lib/wasm/wasm_exec.js" wasm_exec.js

.PHONY: test
test:
	go test ./...

# Regenerate the golden amended certificate used by the test and harnesses.
.PHONY: fixtures
fixtures:
	go run . testdata/stm-tpm-ecc/stmtpmeccroot01.pem \
		testdata/stm-tpm-ecc/stmtpmeccint02.pem > testdata/stm-tpm-ecc/amended.pem
