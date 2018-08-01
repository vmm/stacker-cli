PROJECT_NAME="stacker"

test:
	@go test ./...

build:
	@go build -o bin/stacker ./cmd/stacker

deps:
	@which dep > /dev/null || go get -u github.com/golang/dep/cmd/dep
	@dep ensure

format:
	@which goimports > /dev/null || go get golang.org/x/tools/cmd/goimports
	@echo "--> Running goimports"
	@goimports -w .

clean:
	@rm -rf ./bin/* > /dev/null

.PHONY: test update deps format clean
