IMPORT_PATH := github.com/FiloSottile/tlsbench
V := 1 # print commands and build progress by default
IGNORED_PACKAGES := /vendor/ # space separated patterns

.PHONY: all
all: tlsbench

.PHONY: tlsbench bin/tlsbench
tlsbench: bin/tlsbench
bin/tlsbench: .GOPATH/.ok
	$Q go install $(if $V,-v) $(VERSION_FLAGS) $(IMPORT_PATH)

##### =====> Utility targets <===== #####

.PHONY: clean test list cover format

clean:
	$Q rm -rf bin .GOPATH

test: .GOPATH/.ok
	$Q go test $(if $V,-v) -i -race $(allpackages) # install -race libs to speed up next run
ifndef CI
	$Q go vet $(allpackages)
	$Q GODEBUG=cgocheck=2 go test -race $(allpackages)
else
	$Q go vet $(allpackages) | tee .GOPATH/test/vet.txt
	$Q GODEBUG=cgocheck=2 go test -v -race $(allpackages) | tee .GOPATH/test/output.txt
endif

list: .GOPATH/.ok
	@echo $(allpackages)

cover: bin/gocovmerge .GOPATH/.ok
	$Q rm -f .GOPATH/cover/*.out .GOPATH/cover/all.merged
	$(if $V,@echo "-- go test -coverpkg=./... -coverprofile=.GOPATH/cover/... ./...")
	$Q for MOD in $(allpackages); do \
		go test -coverpkg=`echo $(allpackages)|tr " " ","` \
			-coverprofile=.GOPATH/cover/unit-`echo $$MOD|tr "/" "_"`.out \
			$$MOD 2>&1 | grep -v "no packages being tested depend on" || exit 1; \
	done
	$Q ./bin/gocovmerge .GOPATH/cover/*.out > .GOPATH/cover/all.merged
ifndef CI
	$Q go tool cover -html .GOPATH/cover/all.merged
else
	$Q go tool cover -html .GOPATH/cover/all.merged -o .GOPATH/cover/all.html
endif
	@echo ""
	@echo "=====> Total test coverage: <====="
	@echo ""
	$Q go tool cover -func .GOPATH/cover/all.merged

format: bin/goimports .GOPATH/.ok
	$Q find .GOPATH/src/$(IMPORT_PATH)/ -iname \*.go | grep -v -e "^$$" $(addprefix -e ,$(IGNORED_PACKAGES)) | xargs ./bin/goimports -w

##### =====> Internals <===== #####

VERSION          := $(shell git describe --tags --always --dirty="-dev")
DATE             := $(shell date '+%Y-%m-%d-%H%M UTC')
VERSION_FLAGS    := -ldflags='-X "main.Version=$(VERSION)" -X "main.BuildTime=$(DATE)"'

# cd into the GOPATH to workaround ./... not following symlinks (lazily computed)
allpackages = $(eval allpackages := $(shell ( cd $(CURDIR)/.GOPATH/src/$(IMPORT_PATH) && \
    GOPATH=$(CURDIR)/.GOPATH go list ./... 2>&1 1>&3 | \
    grep -v -e "^$$" $(addprefix -e ,$(IGNORED_PACKAGES)) 1>&2 ) 3>&1 | \
    grep -v -e "^$$" $(addprefix -e ,$(IGNORED_PACKAGES))))$(allpackages)

export GOPATH := $(CURDIR)/.GOPATH

Q := $(if $V,,@)

.GOPATH/.ok:
	$Q mkdir -p "$(dir .GOPATH/src/$(IMPORT_PATH))"
	$Q ln -s ../../../.. ".GOPATH/src/$(IMPORT_PATH)"
	$Q mkdir -p .GOPATH/test .GOPATH/cover
	$Q mkdir -p bin
	$Q ln -s ../bin .GOPATH/bin
	$Q touch $@

.PHONY: bin/gocovmerge bin/goimports
bin/gocovmerge: .GOPATH/.ok
	$Q go install $(IMPORT_PATH)/vendor/github.com/wadey/gocovmerge
bin/goimports: .GOPATH/.ok
	$Q go install $(IMPORT_PATH)/vendor/golang.org/x/tools/cmd/goimports
