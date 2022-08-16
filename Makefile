
# All
all: upstream docker-build

# Build dependencies
install-tools:
	pushd vendor/github.com/onflow/flow-go/crypto && go generate && go build; popd

# Run API service
run:
	go run -v -tags=relic cmd/api-service/main.go

# Build API service
build:
	go build -v -tags=relic -o /app cmd/api-service/main.go

# Test API service
test:
	go test -v -tags=relic ./...

# Run API service in Docker
docker-run: docker-build
	docker run -d --name flow_api_service --rm -p 4900:9000 onflow.org/api-service go run -v -tags=relic cmd/api-service/main.go

# Run build/test/run debug console
debug:
	docker build -t onflow.org/api-service-debug --target build-dependencies .
	docker run -t -i --rm onflow.org/api-service-debug /bin/bash

# Run all unit tests
docker-test: docker-build-test
	docker run --rm onflow.org/api-service-test go test -v -tags=relic ./...

# Build production Docker containers
docker-build:
	docker build -t onflow.org/api-service --target production .
	docker build -t onflow.org/api-service-small --target production-small .
	docker build -t onflow.org/flow-cli --target flow-cli .
	docker build -t onflow.org/flow-e2e-test --target flow-e2e-test .
	docker build -t onflow.org/flow-client-execution --target flow-client .

# Build intermediate build docker container
docker-build-test:
	docker build -t onflow.org/api-service-test --target build-env .

# Run API service attached to Flow localnet network in Docker
docker-test-e2e: clean docker-test-localnet-cleaned

# Stop localnet Flow tests
docker-test-localnet-cleaned: docker-test-localnet
	bash -c 'cd upstream/flow-go/integration/localnet && make stop'

# Run API service attached to localnet in Docker
docker-test-localnet: docker-run-localnet
	# Start a DPS instance
	docker run -d --name localnet_dps --rm -p 127.0.0.1:9555:9000 --link localnet_access_1_1:access --network localnet_default onflow.org/flow-dps-emu
	# Wait for an arbitrary but credible amount of time to start up
	sleep 3
	# Start an API service
	docker run -d --name localnet_flow_api_service --rm -p 127.0.0.1:9500:9000 --network localnet_default \
	--link access_1:access --link localnet_dps:dps onflow.org/api-service go run -v -tags=relic cmd/api-service/main.go \
	--rpc-addr=:9000 \
	--protocol-node-addresses=access:9000 \
	--protocol-node-public-keys=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa \
	--execution-node-addresses=access:9000 \
	--execution-node-public-keys=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa \
	--dps-node-addresses=dps:9000 \
	--dps-node-public-keys=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa \

	# Wait for an arbitrary but credible amount of time to start up
	sleep 10
	# To follow: docker logs -f localnet_flow_api_service
	docker logs localnet_flow_api_service
	# Check latest block: flow -f ./resources/flow-localnet.json -n api blocks get latest
	docker run --rm --link localnet_flow_api_service:flow_api  --network localnet_default \
		onflow.org/flow-e2e-test
	# Run an execution test
	docker run --rm --link localnet_flow_api_service:flow_api  --network localnet_default \
		onflow.org/flow-client-execution
	# Stop the API service created above
	docker stop localnet_flow_api_service
	docker stop localnet_dps

# Run a Flow network in a localnet in Docker
docker-run-localnet: upstream docker-build
	bash -c 'cd upstream/flow-go/integration/localnet && make init && make start'

# Install prerequisites
upstream:
	# Make sure we have the directory for prerequisites
	mkdir -p upstream

	# We might want to use testnet
	git clone https://github.com/GetElastech/flow-dps-emu.git upstream/flow-dps-emu || true

	# Install its prerequisites
	bash -c 'cd upstream/flow-dps-emu && make'

	# Let's link https://github.com/onflow/flow-go.git instead of cloning
	bash -c 'cd upstream && ln -s flow-dps-emu/upstream/flow-go .'

	# Get crypto libs
	bash -c 'cd upstream/flow-go && make install-tools'

# Clean all images and unused containers
clean:
	bash -c 'docker stop localnet_dps || true'
	bash -c 'docker stop localnet_flow_api_service || true'
	bash -c '! test -d upstream/flow-go/integration/localnet || (cd upstream/flow-go/integration/localnet && make stop || true)'
	rm -rf upstream
	docker system prune -a -f

