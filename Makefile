.PHONY: build test test-cover lint docker-build clean run

build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o build/iptv-builder ./cmd/iptv-builder

test:
	go test -v -race -cover -count=1 ./...

test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

lint:
	golangci-lint run ./...

run: build
	./build/iptv-builder --config-dir ./configs --output-dir ./output --cache-dir ./cache

docker-build:
	docker build -t iptv-builder:latest .

clean:
	rm -rf build/ output/ cache/ coverage.out coverage.html
