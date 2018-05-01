IMPORT_PATH := github.com/FiloSottile/mostly-harmless/covfefe

.PHONY: covfefe enroll rescan
covfefe: .GOPATH/.ok
	GOPATH="$(PWD)/.GOPATH" go install -v $(IMPORT_PATH)/cmd/covfefe
enroll: .GOPATH/.ok
	GOPATH="$(PWD)/.GOPATH" go install -v $(IMPORT_PATH)/cmd/enroll
rescan: .GOPATH/.ok
	GOPATH="$(PWD)/.GOPATH" go install -v $(IMPORT_PATH)/cmd/rescan

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
