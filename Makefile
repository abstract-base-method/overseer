.PHONY: init
init:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

.PHONY: generate
generate:
	buf generate

.PHONY: test
test: generate
	go test -v ./...

.PHONY: clean
clean:
	rm -rf build/
	rm -rf ollama/