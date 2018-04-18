IMPORT_PATH := github.com/FiloSottile/mostly-harmless/dcbot

.PHONY: build
build: .GOPATH/.ok
	GOPATH="$(PWD)/.GOPATH" go install -v $(IMPORT_PATH)

.PHONY: clean
clean:
	rm -rf bin .GOPATH

.GOPATH/.ok:
	mkdir -p ".GOPATH/src/$(IMPORT_PATH)"
	rmdir ".GOPATH/src/$(IMPORT_PATH)"
	ln -s ../../../../.. ".GOPATH/src/$(IMPORT_PATH)"
	mkdir -p .GOPATH/test .GOPATH/cover
	mkdir -p bin
	ln -s ../bin .GOPATH/bin
	touch $@
