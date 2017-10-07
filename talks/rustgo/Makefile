IMPORT_PATH      := github.com/FiloSottile/mostly-harmless/talks/rustgo
INSTALL_PATH     := $(shell go env GOPATH)/pkg/$(shell go env GOOS)_$(shell go env GOARCH)/$(IMPORT_PATH)

multiplication/multiplication.a: multiplication/mul.go multiplication/mul.o multiplication/libmultiplication.o
		go tool compile -N -l -o $@ -p main -pack multiplication/mul.go
		go tool pack r $@ multiplication/mul.o multiplication/libmultiplication.o

multiplication/libmultiplication.o: target/release/libmultiplication.a
ifeq ($(shell go env GOOS),darwin)
		ld -r -o $@ -arch x86_64 -u _multiply_two $^
else
		ld -r -o $@ --gc-sections -u multiply_two $^
endif

target/release/libmultiplication.a: src/* Cargo.toml
		cargo build --release

multiplication/mul.o: multiplication/mul.s
		go tool asm -I "$(shell go env GOROOT)/pkg/include" -o $@ $^

.PHONY: install uninstall
install: multiplication/multiplication.a
		mkdir -p "$(INSTALL_PATH)"
		cp multiplication/multiplication.a "$(INSTALL_PATH)"
uninstall:
		rm -f "$(INSTALL_PATH)/multiplication.a"

.PHONY: clean
clean:
		rm -rf multiplication/*.[oa] target
