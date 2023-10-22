
# add git commit hash and date in app binary
BUILD_HASH=`git rev-parse --short HEAD`
BUILD_DATE=`date +%FT%T%z`
LDFLAGS=-ldflags "-w -s -X main.VersionHash=${BUILD_HASH} -X main.BuildDate=${BUILD_DATE}"

# BINARY_NAME
BINARY_NAME=cctv-backup
PROJECT_ID?=tdshop-data-internal
REPO?=tdshop-dp-docker-artifacts

DOCKER_REGISTRY=asia-southeast1-docker.pkg.dev/${PROJECT_ID}/${REPO}
VERSION?=0.0.0

all: help

## help: print this help message
help:
	@echo "Usage:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-30s\033[0m %s\n", $$1, $$2}'
.PHONY: help

# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #

tidy: ## Format code and tidy modfile
	@go fmt ./...	
	@go mod tidy -v

.PHONY: tidy

audit: tidy ## Static code check
	@echo "Static check"
	@go vet ./...
	@go mod verify
	@go run honnef.co/go/tools/cmd/staticcheck@latest -checks=all,-ST1000,-U1000 ./...
    @go run golang.org/x/vuln/cmd/govulncheck@latest ./...
    @go test -race -buildvcs -vet=off ./...
.PHONY: audit

# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

clean:
 go clean
 rm target/${BINARY_NAME}-darwin
 rm target/${BINARY_NAME}-linux
.PHONY: clean

test: tidy ## Run all unit test
	
.PHONY: test

build: tidy ## Build binary file
	GOARCH=amd64 GOOS=darwin go build -o target/${BINARY_NAME}-darwin main.go
	GOARCH=amd64 GOOS=linux go build -o target/${BINARY_NAME}-linux main.go

.PHONY: build

run: tidy ## Run app
	@echo "run build"
.PHONY: run

# ==================================================================================== #
# DOCKER BUILD
# ==================================================================================== #

docker-build: ## Use the dockerfile to build the container
	docker build --rm --tag $(BINARY_NAME) .
.PHONY: docker-build

docker-release: ## Release the container with tag latest and version
	docker tag $(BINARY_NAME) $(DOCKER_REGISTRY)/$(BINARY_NAME):latest
	docker tag $(BINARY_NAME) $(DOCKER_REGISTRY)/$(BINARY_NAME):$(VERSION)
	# Push the docker images
	docker push $(DOCKER_REGISTRY)/$(BINARY_NAME):latest
	docker push $(DOCKER_REGISTRY)/$(BINARY_NAME):$(VERSION)
