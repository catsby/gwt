default: build

build:
    go build -o gwt .

test:
    go test ./...

test-verbose:
    go test ./... -v

vet:
    go vet ./...

fmt:
    gofmt -w .

lint: vet fmt

install:
    go install .

clean:
    rm -f gwt
    go clean ./...

check: lint test
