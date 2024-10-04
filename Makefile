GITCOMMIT := $(shell git rev-parse HEAD)
GITDATE := $(shell git show -s --format='%ct')

LDFLAGSSTRING +=-X main.GitCommit=$(GITCOMMIT)
LDFLAGSSTRING +=-X main.GitDate=$(GITDATE)
LDFLAGS := -ldflags "$(LDFLAGSSTRING)"

multichain-transaction-syncs:
	env GO111MODULE=on go build -v $(LDFLAGS) ./cmd/multichain-transaction-syncs

clean:
	rm multichain-transaction-syncs

test:
	go test -v ./...

lint:
	golangci-lint run ./...

.PHONY: \
	multichain-transaction-syncs \
	bindings \
	bindings-scc \
	clean \
	test \
	lint