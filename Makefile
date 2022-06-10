
# Build dependencies
.PHONY: install-tools
install-tools:
	pushd vendor/github.com/onflow/flow-go/crypto && go generate && go build; popd

# Run API service
.PHONY: run
run:
	go run -v -tags=relic cmd/api-service/main.go

# Build API service
.PHONY: build
build:
	go build -v -tags=relic -o /app cmd/api-service/main.go

# Test API service
.PHONY: test
test:
	go test -v -tags=relic ./...

# Run API service in Docker
.PHONY: docker-run
docker-run: docker-build
	docker run -d --name flow_api_service --rm -p 4900:9000 onflow.org/api-service go run -v -tags=relic cmd/api-service/main.go

# Run build/test/run debug console
.PHONY: debug
debug:
	docker build -t onflow.org/api-service-debug --target build-dependencies .
	docker run -t -i --rm onflow.org/api-service-debug /bin/bash

# Run all tests
.PHONY: docker-test
docker-test: docker-build-test
	docker run --rm onflow.org/api-service go test -v -tags=relic ./...

# Build production Docker container
.PHONY: docker-build
docker-build:
	docker build -t onflow.org/api-service --target production .
	docker build -t onflow.org/api-service-small --target production-small .
	docker build -t onflow.org/flow-cli --target flow-cli .
	docker build -t onflow.org/flow-e2e-test --target flow-e2e-test .

# Build intermediate build docker container
.PHONY: docker-build-test
docker-build-test:
	docker build -t onflow.org/api-service --target build-env .

# Clean all
.PHONY: docker-clean
docker-clean:
	docker system prune -a

# Run API service attached to localnet in Docker
.PHONY: docker-test-e2e
docker-test-e2e: docker-test-localnet-cleaned

# Stop localnet Flow tests
.PHONY: docker-test-localnet-cleaned
docker-test-localnet-cleaned: docker-test-localnet
	bash -c 'cd upstream/integration/localnet && make stop'

# Run API service attached to localnet in Docker
.PHONY: docker-test-localnet
docker-test-localnet: docker-run-localnet
	docker run -d --name localnet_flow_api_service --rm -p 127.0.0.1:9500:9000 --network localnet_default \
	--link access_1:access onflow.org/api-service go run -v -tags=relic cmd/api-service/main.go \
	--protocol-node-addresses=access:9000 --execution-node-addresses=access:9000 \
	--protocol-node-public-keys=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa \
	--execution-node-public-keys=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa --rpc-addr=:9000
	sleep 10
	# To follow: docker logs -f localnet_flow_api_service
	docker logs localnet_flow_api_service
	# Check latest block: flow -f ./flow-localnet.json -n api blocks get latest
	docker run --rm --link localnet_flow_api_service:flow_api  --network localnet_default \
		onflow.org/flow-e2e-test
	docker stop localnet_flow_api_service

# Run API service attached to localnet in Docker
.PHONY: docker-run-localnet
docker-run-localnet: docker-build
	# We might want to use testnet
	git clone https://github.com/onflow/flow-go.git upstream || true
	# git checkout e4b4451c233628969ee321dfd5c0b19a0152fe79
	bash -c 'cd upstream && make install-tools'
	bash -c 'cd upstream/integration/localnet && make init && make start'

