default: build

build:
	go build ./...

test:
	go test ./... -count=1 -v

cover:
	go test ./... -count=1 -cover

vet:
	go vet ./...

fmt:
	gofmt -s -w .

fmtcheck:
	@gofmt -s -l . | grep -v vendor | tee /dev/stderr | read && echo "gofmt check failed" && exit 1 || true

.PHONY: build test cover vet fmt fmtcheck
