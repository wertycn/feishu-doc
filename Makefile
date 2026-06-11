BINARY := feishu-doc
VERSION := 0.1.0
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

.PHONY: build test clean release

build:
	go build $(LDFLAGS) -o bin/$(BINARY) ./cmd/feishu-doc

test:
	go test ./...

clean:
	rm -rf bin/ dist/

release:
	$(eval VERSION := $(or $(V),$(VERSION)))
	mkdir -p dist
	@for os in linux darwin; do \
		for arch in amd64 arm64; do \
			echo "Building $$os/$$arch..."; \
			CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch \
				go build $(LDFLAGS) -o dist/$(BINARY)-$(VERSION)-$$os-$$arch ./cmd/feishu-doc; \
		done; \
	done
	@for arch in amd64 arm64; do \
		echo "Building windows/$$arch..."; \
		CGO_ENABLED=0 GOOS=windows GOARCH=$$arch \
			go build $(LDFLAGS) -o dist/$(BINARY)-$(VERSION)-windows-$$arch.exe ./cmd/feishu-doc; \
	done
	ls -lh dist/
