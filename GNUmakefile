all: kubemove-cli datasync engine pair

PACKAGES = $(shell go list ./... | grep -v 'vendor')

format:
	@echo "--> Running go fmt"
	@go fmt $(PACKAGES)

vet:
	go vet ${PACKAGES}

golint:
	@gometalinter --install
	@gometalinter --vendor --disable-all -E errcheck -E misspell ./...

kubemove-cli:
	@echo "Building kubemove-cli"
	@rm -rf _output/bin/kubemove
	@go build -o _output/bin/kubemove cmd/kubemove/main.go
	@echo "Done"

datasync:
	@echo "Building kubemove-datasync"
	@rm -rf _output/bin/datasync
	@go build -o _output/bin/datasync cmd/datasync/main.go
	@echo "Done"

engine:
	@echo "Building kubemove-engine"
	@rm -rf _output/bin/kengine
	@go build -o _output/bin/kengine cmd/engine/main.go
	@echo "Done"

pair:
	@echo "Building kubemove-pair"
	@rm -rf _output/bin/kpair
	@go build -o _output/bin/kpair cmd/pair/main.go
	@echo "Done"

clean:
	@echo "Removing old binaries"
	@rm -rf _output
	@echo "Done"
