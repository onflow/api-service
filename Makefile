
# All
all: docker-test-e2e docker-build-test

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

# Build intermediate build docker container
docker-build-test:
	docker build -t onflow.org/api-service-test --target build-env .

# Run API service attached to Flow localnet network in Docker
docker-test-e2e: docker-test-localnet-cleaned

# Stop localnet Flow tests
docker-test-localnet-cleaned: docker-test-localnet
	bash -c 'cd upstream/flow-go/integration/localnet && make stop'

# Run API service attached to localnet in Docker
docker-test-localnet: docker-run-localnet
	docker run -d --name localnet_flow_api_service --rm -p 127.0.0.1:9500:9000 --network localnet_default \
	--link access_1:access onflow.org/api-service go run -v -tags=relic cmd/api-service/main.go \
	--protocol-node-addresses=access:9000 \
	--protocol-node-public-keys=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa \
	--execution-node-addresses=access:9000 \
	--execution-node-public-keys=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa \
	--dps-node-addresses=dps:9000 \
	--dps-node-public-keys=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa --rpc-addr=:9000
	# Wait for an arbitrary but credible amount of time to start up
	sleep 10
	# To follow: docker logs -f localnet_flow_api_service
	docker logs localnet_flow_api_service
	# Check latest block: flow -f ./resources/flow-localnet.json -n api blocks get latest
	docker run --rm --link localnet_flow_api_service:flow_api  --network localnet_default \
		onflow.org/flow-e2e-test
	# Stop the API service created above
	docker stop localnet_flow_api_service

# Run a Flow network in a localnet in Docker
docker-run-localnet: upstream docker-build
	bash -c 'cd upstream/flow-go/integration/localnet && make init && make start'
	bash -c 'cd upstream/flow-dps-emu && make'
	docker run -d --name dps --rm --link access_1:access --network localnet_default onflow.org/flow-dps-emu

# Install prerequisites
upstream:
	mkdir -p upstream
	# We might want to use testnet
	git clone https://github.com/GetElastech/flow-dps-emu.git upstream/flow-dps-emu || true
	# Install its prerequisites
	bash -c 'cd upstream/flow-dps-emu && make upstream'

	#git clone https://github.com/onflow/flow-go.git upstream/flow-go || true
	# Instead of cloning let's link
	bash -c 'cd upstream && ln -s flow-dps-emu/upstream/flow-go .'

	# Get crypto libs
	bash -c 'cd upstream/flow-go && make install-tools'

# Clean all images and unused containers
clean:
	docker system prune -a -f

