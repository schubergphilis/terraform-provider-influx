TEST?=./...
PKG_NAME=influx

default: build

build:
	go install

fmtcheck:
	@sh -c "'$(CURDIR)/scripts/gofmtcheck.sh'"

test: fmtcheck
	go test $(TEST) -timeout=30s -parallel=4


test-compile:
	@if [ "$(TEST)" = "./..." ]; then \
		echo "ERROR: Set TEST to a specific package. For example,"; \
		echo "  make test-compile TEST=./$(PKG_NAME)"; \
		exit 1; \
	fi
	go test -c $(TEST) $(TESTARGS)

testacc:
	TF_SCHEMA_PANIC_ON_ERROR=1 \
	TF_ACC=1 go test $(TEST) -v $(TESTARGS) -timeout 10m

.PHONY: build fmtcheck test test-compile testacc
