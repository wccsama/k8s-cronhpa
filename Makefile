# Image URL to use all building/pushing image targets
HUB ?= HUB

all: build

# Build manager binary
build: fmt vet
	GOOS=linux GOARCH=amd64 go build

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Build the docker image
docker-build:
	docker build . -t ${HUB}

# Push the docker image
docker-push:
	docker push ${HUB}

make-image: build docker-build docker-push
