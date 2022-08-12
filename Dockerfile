# NOTE: Must be run in the context of the repo's root directory

## (1) Download a suitable version of Go
FROM golang:1.18 AS build-setup

# Add optional items like apt install -y make cmake gcc g++
RUN apt update && apt install -y cmake make gcc g++

## (2) Build the app binary
FROM build-setup AS build-dependencies

# Cache gopath dependencies for faster builds
# Newer projects should opt for go mod vendor for reliability and security
RUN mkdir /app
RUN mkdir /app/src
COPY src/go.mod /app/src
COPY src/go.sum /app/src

# FIX: This generates code marked by `go:build relic` and `+build relic`. See `combined_verifier_v3.go`.
# FIX: This is not needed, if vendor/ is used
# NOTE: crypto@v0.24.4 is the latest stable version from flow-go
WORKDIR /app/src
RUN go mod download
RUN go mod download github.com/onflow/flow-go/crypto@v0.24.3
RUN cd $GOPATH/pkg/mod/github.com/onflow/flow-go/crypto@v0.24.3 && go generate && go build

## (3) Build the app binary
FROM build-dependencies AS build-env

COPY src /app/src
WORKDIR /app/src

# Fix: Make sure no further steps update modules later, so that we can debug regressions by keeping the Docker image
RUN go mod vendor
RUN cp -R $GOPATH/pkg/mod/github.com/onflow/flow-go/crypto@v0.24.3/* /app/src/vendor/github.com/onflow/flow-go/crypto
RUN ls /app/src/vendor/github.com/onflow/flow-go/crypto/relic

# FIX: Without -tags=relic we get undefined: "github.com/onflow/flow-go/consensus/hotstuff/verification".NewCombinedVerifier
RUN go build -v -tags=relic -o /app cmd/api-service/main.go

# Build environment for go build -tags=relic cmd/api-service/main.go
CMD /bin/bash

## (4) Add the statically linked binary to a distroless image
FROM build-env as production

WORKDIR /app/src
COPY --from=build-env /app/main /app/main

CMD ["go", "run", "-tags=relic", "cmd/api-service/main.go"]

## (5) Add the statically linked binary to a distroless image
FROM golang:1.18 as production-small

RUN rm -rf /go
RUN rm -rf /app
RUN rm -rf /usr/local/go
COPY --from=production /app/main /bin/main

CMD ["/bin/main"]

## (6) Flow client build environment for checking backward compatibility
FROM golang:1.18 as build-cli-env

RUN git clone https://github.com/onflow/flow-cli.git /flow-cli
WORKDIR /flow-cli

# FIX: Let's stick to v0.34.0 to make sure we are backward compatible with legacy clients end to end.
RUN git checkout 6c240a76ec2bb5d5685afeb0898eed0ea1bd0059
RUN go mod download
# FIX: make sure no further steps update modules later, so that we can debug regressions
RUN go mod vendor

## (7) Flow client build for checking backward compatibility
FROM build-cli-env as build-cli

WORKDIR /flow-cli
# FIX: Let's not gamble and stick to v0.34.0. Backward compatibility can be checked this way.
# FIX: See git checkout in build-cli-env
RUN VERSION=v0.34.0 \
	go build \
	-trimpath \
	-ldflags \
	"-X github.com/onflow/flow-cli/build.commit=6c240a76ec2bb5d5685afeb0898eed0ea1bd0059 -X github.com/onflow/flow-cli/build.semver=v0.34.0" \
	./cmd/flow/main.go

RUN ./main version

## (8) Very lean Flow client image
FROM golang:1.18 as flow-cli

RUN rm -rf /go
RUN rm -rf /app
RUN rm -rf /usr/local/go
COPY --from=build-cli /flow-cli/main /bin/flow

CMD ["/bin/bash"]

## (9) End to end testing client calls
FROM flow-cli as flow-e2e-test

COPY ./resources/flow-localnet.json /root/flow-localnet.json
WORKDIR /root
CMD flow -f /root/flow-localnet.json -n flow_api blocks get latest

FROM golang:1.18 as flow-client

#RUN bash -c 'echo {\"networks\": {\"access\": \"127.0.0.1:3569\", \"observer\": \"127.0.0.1:3573\"}} >/go/flow-localnet.json'
RUN bash -c 'echo {\"networks\": {\"api\":\"127.0.0.1:9500\", \"flow_api\":\"flow_api:9000\", \"access\": \"127.0.0.1:3571\", \"observer\": \"127.0.0.1:3569i\"}} >/go/flow-localnet.json'
RUN bash -c 'printf "pub fun main(greeting: String, who: String): String\n{\n return greeting.concat(\" \").concat(who)\n}\n" >/go/hello.cdc'

WORKDIR /go
RUN curl -L https://github.com/onflow/flow-cli/archive/refs/tags/v0.36.2.tar.gz | tar -xzv
RUN cd flow-cli-0.36.2 && go mod download
RUN cd flow-cli-0.36.2 && make
RUN /go/flow-cli-0.36.2/cmd/flow/flow version
RUN cp /go/flow-cli-0.36.2/cmd/flow/flow /go/flow

CMD ["/go/flow", "-f", "/go/flow-localnet.json", "-n", "flow_api", "scripts", "execute", "hello.cdc", "Hello", "World!"]

