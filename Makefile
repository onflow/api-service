
# Run API service
.PHONY: run
run: build
	docker run -t -i --rm onflow.org/api-service

# Run build/test/run debug console
.PHONY: debug
docker-debug: build-intermediate
	docker run -t -i --rm onflow.org/api-service-build /bin/bash

# Run all tests
.PHONY: test
test: build-intermediate
	docker run -t -i --rm onflow.org/api-service-build go test -v -tags=relic ./...

# Build production Docker container
.PHONY: build
build:
	docker build -t onflow.org/api-service --target production .

# Build intermediate build docker container
.PHONY: build-intermediate
build-intermediate:
	docker build -t onflow.org/api-service-build --target build-env .
