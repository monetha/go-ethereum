PKGS ?= $(shell glide novendor)
BENCH_FLAGS ?= -benchmem

.PHONY: all
all: lint test

.PHONY: dependencies
dependencies:
	@echo "Installing Glide and locked dependencies..."
	glide --version || go get -u -f github.com/Masterminds/glide
	glide install
	@echo "Installing goimports..."
	go install ./vendor/golang.org/x/tools/cmd/goimports
	@echo "Installing golint..."
	go install ./vendor/golang.org/x/lint/golint
	@echo "Installing gosimple..."
	go install ./vendor/honnef.co/go/tools/cmd/gosimple
	@echo "Installing unused..."
	go install ./vendor/honnef.co/go/tools/cmd/unused
	@echo "Installing staticcheck..."
	go install ./vendor/honnef.co/go/tools/cmd/staticcheck

.PHONY: lint
lint:
	@echo "Checking formatting..."
	@gofiles=$$(go list -f {{.Dir}} $(PKGS) | grep -v mock) && [ -z "$$gofiles" ] || unformatted=$$(for d in $$gofiles; do goimports -l $$d/*.go; done) && [ -z "$$unformatted" ] || (echo >&2 "Go files must be formatted with goimports. Following files has problem:\n$$unformatted" && false)
	@echo "Checking vet..."
	@go vet $(PKG_FILES)
	@echo "Checking simple..."
	@gosimple $(PKG_FILES)
	@echo "Checking unused..."
	@unused $(PKG_FILES)
	@echo "Checking staticcheck..."
	@staticcheck $(PKG_FILES)
	@echo "Checking lint..."
	@$(foreach dir,$(PKGS),golint $(dir);)

.PHONY: test
test:
	go test -timeout 20s -race -v $(PKGS)

.PHONY: cover
cover:
	./cover.sh $(PKGS)

.PHONY: bench
BENCH ?= .
bench:
	$(foreach pkg,$(PKGS),go test -bench=$(BENCH) -run="^$$" $(BENCH_FLAGS) $(pkg);)

.PHONY: fmt
fmt:
	@echo "Formatting files..."
	@gofiles=$$(go list -f {{.Dir}} $(PKGS) | grep -v mock) && [ -z "$$gofiles" ] || for d in $$gofiles; do goimports -l -w $$d/*.go; done

