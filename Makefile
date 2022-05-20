
.PHONY: build
run: docker-build

.PHONY: run
run: docker-run

.PHONY: test
test: docker-test

# Build all executables
.PHONY: docker-run
docker-run: docker-build
	docker run -t -i --rm onflow.org/api-service

# Run build/test/run debug console
.PHONY: docker-debug
docker-debug: docker-build-intermediate
	docker run -t -i --rm onflow.org/api-service-build /bin/bash

# Run all tests
.PHONY: docker-test
docker-test:
	docker build -t onflow.org/api-service-test --target test .
	docker run -t -i --rm onflow.org/api-service-test go test -v ./...

# Build production Docker container
.PHONY: docker-build
docker-build:
	docker build -t onflow.org/api-service --target production .

# Build intermediate build docker container
.PHONY: docker-build-intermediate
docker-build-intermediate:
	docker build -t onflow.org/api-service-build --target build-env .
