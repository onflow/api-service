## Flow API service

This is an interface service to the Flow network. You can use it as a thin endpoint to view and change network data.

# License

(LICENSE)[LICENSE]

# Requirements

The API service should run in a container with 1 vCPU and 1GB of assigned memory.

# Usage

Building the project creates the container image `onflow.org/api-service`.

```
make docker-build
```

Run all tests that are part of the code submit process. This may run up to 20 minutes.

```
make docker-test-e2e
```

Run the image built.

```
docker run -d --name flow_api_service --rm -p 127.0.0.1:9500:9000 \
--link access_1:access onflow.org/api-service go run -v -tags=relic cmd/api-service/main.go \
--protocol-node-addresses=access_1.onflow.org:9000 \
--protocol-node-public-keys=d63f05927bd0379fac99ebb27b696592 \
--execution-node-addresses=execution_1.onflow.org:9000 \
--execution-node-public-keys=d62d3f4c038a366a8bcdc883e979b6db --rpc-addr=:9000
--dps-node-addresses=dps_1.onflow.org:9000 \
--dps-node-public-keys=d62d3f4c038a366a8bcdc883e979b6db --rpc-addr=:9000
```

# Behavior

The API service connects to upstream endpoints. Access nodes and observer nodes provide protocol and execution endpoints of current sporks.
DPS nodes provide both protocol and execution access to earlier sporks. Calls default to the current spork first.
The service is designed in a way so that it runs with just command line settings, no additional configuration files are needed for the container.

Argument Details

**rpc-addr** The local address of the GRPC server that the API service is listening on.
**flow-api-timeout** The TCP timeout of the Flow API gRPC socket.
**protocol-node-addresses** A comma separated list of network addresses of the bootstrap access nodes e.g. access-001.mainnet.flow.org:9653,access-002.mainnet.flow.org:9653
**protocol-node-public-keys** A comma separated list of networking public keys of the bootstrap access nodes (in the same order as the bootstrap node addresses) e.g. \"d57a5e9c5.....\",\"44ded42d....\" Unsecure channel is used, if none is specified.
**execution-node-addresses** A comma separated list of network addresses of the bootstrap execution nodes e.g. access-001.mainnet.flow.org:9653,access-002.mainnet.flow.org:9653
**execution-node-public-keys** A comma separated list of networking public keys of the bootstrap execution nodes (in the same order as the bootstrap node addresses) e.g. \"d57a5e9c5.....\",\"44ded42d....\" Unsecure channel is used, if none is specified.
**flow-dps-node-addresses** A comma separated list of network addresses of the bootstrap DPS nodes e.g. access-001.mainnet.flow.org:9653,access-002.mainnet.flow.org:9653
**flow-dps-publish-public-keys** A comma separated list of networking public keys of the bootstrap DPS nodes (in the same order as the bootstrap node addresses) e.g. \"d57a5e9c5.....\",\"44ded42d....\" Unsecure channel is used, if none is specified.
**cache-size** The parameter specifies the maximum cache size for register reads in flow-dps in bytes
