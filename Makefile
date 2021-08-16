GENERATE_SIMPLE:=\
	$(NULL)
GENERATE_SPECIAL:=\
	./swagger-ui/ui/swagger-ui-bundle.js \
	$(NULL)
GENERATED_FILES:=\
	$(GENERATE_SIMPLE) \
	$(GENERATE_SPECIAL) \
	$(NULL)

ifneq ($(CIRCLE_PROJECT_REPONAME),)
REPONAME:=$(CIRCLE_PROJECT_REPONAME)
else
REPONAME:=$(notdir $(CURDIR))
endif

# always test with race and coverage, we'll run vet separately.
TESTARGS:=-vet=off -race -cover -coverpkg=./...

GOIMPORTSARGS:=-local github.com/6RiverSystems,go.6river.tech

# default `make` invocation is to run the full generate/build/lint/test sequence
default: compile-code test
.PHONY: default

generate: $(GENERATED_FILES)
.PHONY: generate

$(GENERATE_SIMPLE): %.go:
	go generate -x ./$(dir $@)
	gofmt -l -s -w ./$(dir $@)
	go run golang.org/x/tools/cmd/goimports -l -w $(GOIMPORTSARGS) ./$(dir $@)

./swagger-ui/ui/swagger-ui-bundle.js: ./swagger-ui/generate.go ./swagger-ui/get-ui/get-swagger-ui.go
	go generate -x ./swagger-ui
	gofmt -l -s -w ./swagger-ui
	go run golang.org/x/tools/cmd/goimports -l -w $(GOIMPORTSARGS) ./swagger-ui

# this requires at least go 1.16.5 to avoid mucking up go.mod/go.sum
get:
	go mod download -x
	go mod verify

install-ci-tools:
# tools only needed in CI
	go install gotest.tools/gotestsum
tools:
	mkdir -p ./tools
	GOBIN=$(PWD)/tools go install github.com/golangci/golangci-lint/cmd/golangci-lint
.PHONY: get install-ci-tools tools

fmt:
	gofmt -l -s -w .
	go run golang.org/x/tools/cmd/goimports -l -w $(GOIMPORTSARGS) .
# format just the generated files
fmt-generated: $(GENERATED_FILES)
	git ls-files --exclude-standard --others --ignored -z | grep -z '\.go$$' | xargs -0 gofmt -l -s -w
	git ls-files --exclude-standard --others --ignored -z | grep -z '\.go$$' | xargs -0 go run golang.org/x/tools/cmd/goimports -l -w $(GOIMPORTSARGS)
.PHONY: fmt fmt-generated

# <() construct requires bash
lint : SHELL=/bin/bash
lint:
# use inverted grep exit code to both print results and fail if there are any
# fgrep -xvf... is used to exclude exact matches from the list of git ignored files
	! gofmt -l -s . | fgrep -xvf <( git ls-files --exclude-standard --others --ignored ) | grep .
	! go run golang.org/x/tools/cmd/goimports -l $(GOIMPORTSARGS) . | fgrep -xvf <( git ls-files --exclude-standard --others --ignored ) | grep .
	go run github.com/golangci/golangci-lint/cmd/golangci-lint run
	go run github.com/google/addlicense -c '6 River Systems' -l mit -skip css -skip js -skip yml -skip html -check .
.PHONY: lint

compile: compile-code compile-tests
compile-code: generate
	go build -v ./...
# this weird hack makes go compile the tests but not run them. basically this
# seeds the build cache and gives us any compile errors. unforunately it also
# prints out test-like output, so we have to hide that with some grep.
# PIPESTATUS requires bash
compile-tests : SHELL = /bin/bash
compile-tests: generate
	go test $(TESTARGS) -run='^$$' ./... | grep -v '\[no test' ; exit $${PIPESTATUS[0]}
.PHONY: compile compile-code compile-tests

# paranoid: always test with the race detector
test: lint vet test-go
vet:
	go vet ./...
test-go:
	go test $(TESTARGS) -coverprofile=coverage.out ./...
test-go-ci-split:
# this target assumes some variables set on the make command line from the CI
# run, and also that gotestsum is installed, which is not handled by this
# makefile, but instead by the CI environment
	gotestsum --format standard-quiet --junitfile $(TEST_RESULTS)/gotestsum-report.xml -- $(TESTARGS) -coverprofile=${TEST_RESULTS}/coverage.out $(PACKAGE_NAMES)
.PHONY: test vet test-go test-go-ci-split

clean-generated:
	rm -rf $(GENERATED_FILES) ./swagger-ui/ui/ .version
clean: clean-generated
	rm -rf bin/ coverage.out coverage.html
.PHONY: clean clean-generated
