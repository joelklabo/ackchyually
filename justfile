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
  go install ./cmd/ackchyually
  @echo "which ackchyually"
  @which ackchyually || true
  @echo "$(go env GOPATH)/bin/ackchyually version"
  @$(go env GOPATH)/bin/ackchyually version
  @echo "$(go env GOPATH)/bin/ackchyually shim doctor"
  @$(go env GOPATH)/bin/ackchyually shim doctor || true

eval-helpcount:
  go run ./cmd/ackchyually-eval

site-sync-install:
  cp scripts/install.sh site/install.sh
