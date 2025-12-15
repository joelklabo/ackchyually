set dotenv-load := false

default:
  @just -l

fmt:
  gofmt -w .

test:
  go test ./...

test-pty:
  go test ./... -run TestPTY -count=1

cover:
  go test ./... -coverprofile=coverage.out -covermode=atomic
  go tool cover -func=coverage.out | tail -n 1

lint:
  golangci-lint run

bench:
  go test ./... -run '^$' -bench . -benchmem -count=1

build:
  go build ./cmd/ackchyually

install-local:
  go install ./cmd/ackchyually

dev-local:
  @go install ./cmd/ackchyually
  @echo ""
  @echo "ackchyually (local build)"
  @echo ""
  @echo "which ackchyually"
  @which ackchyually || true
  @echo ""
  @echo "ackchyually version"
  @$(go env GOPATH)/bin/ackchyually version
  @echo ""
  @echo "ackchyually shim doctor"
  @$(go env GOPATH)/bin/ackchyually shim doctor || true

eval-helpcount:
  go run ./cmd/ackchyually-eval

eval-toptools-dry:
  go run ./cmd/ackchyually-eval-toptools -dry-run

eval-toptools:
  @echo "WARNING: this is a large smoke test. Add -install to install missing Homebrew formulae."
  go run ./cmd/ackchyually-eval-toptools

eval-toptools-install:
  @echo "WARNING: this will install many Homebrew formulae (slow/expensive)."
  go run ./cmd/ackchyually-eval-toptools -install

site-sync-install:
  cp scripts/install.sh site/install.sh
