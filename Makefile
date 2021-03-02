$(GOPATH)/bin/golangci-lint:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b `go env GOPATH`/bin v1.36.0

lint: $(GOPATH)/bin/golangci-lint
	$(GOPATH)/bin/golangci-lint run --fix --verbose --concurrency 4 --timeout 5m

.PHONY: bin/image-cache-daemon
bin/image-cache-daemon:
	GO_ENABLED=0 go build -ldflags="-w -s" -o bin/image-cache-daemon main.go

bin/warden: warden/warden.c
	mkdir -p bin
	gcc -Os -Wall -Werror -static warden/warden.c -o bin/warden