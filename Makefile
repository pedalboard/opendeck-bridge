BINARY = opendeck-bridge

.PHONY: build build-arm64 clean

build:
	go build -o $(BINARY) .

build-arm64:
	CGO_ENABLED=1 CC=aarch64-linux-gnu-gcc GOOS=linux GOARCH=arm64 go build -o $(BINARY)-arm64 .

clean:
	rm -f $(BINARY) $(BINARY)-arm64
