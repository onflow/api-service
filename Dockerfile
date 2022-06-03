# NOTE: Must be run in the context of the repo's root directory

## (1) Download a suitable version of Go
# FIX: We use 1.17 due to unsupported modules
# FIX: github.com/lucas-clemente/quic-go@v0.24.0/internal/qtls/go118.go:6:13:
# FIX:   cannot use "quic-go doesn't build on Go 1.18 yet."
FROM golang:1.17 AS build-setup

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
WORKDIR /app/src
RUN go mod download
RUN go mod download github.com/onflow/flow-go/crypto@v0.24.3
RUN cd $GOPATH/pkg/mod/github.com/onflow/flow-go/crypto@v0.24.3 && go generate && go build

# FIX: Devs should review all what they use to limit build time
RUN cat go.sum

## (3) Build the app binary
FROM build-dependencies AS build-env

COPY src /app/src
WORKDIR /app/src

# Fix: make sure no further steps update modules later, so that we can debug regressions
RUN go mod vendor
RUN cp -R $GOPATH/pkg/mod/github.com/onflow/flow-go/crypto@v0.24.3/* /app/src/vendor/github.com/onflow/flow-go/crypto
RUN ls /app/src/vendor/github.com/onflow/flow-go/crypto/relic

# FIX: Without -tags=relic we get undefined: "github.com/onflow/flow-go/consensus/hotstuff/verification".NewCombinedVerifier
RUN go build -v -tags=relic -o /app cmd/api-service/main.go

CMD /bin/bash

## (5) Add the statically linked binary to a distroless image
FROM build-env as production

WORKDIR /app/src
COPY --from=build-env /app/main /app/main

CMD ["go", "run", "-tags=relic", "cmd/api-service/main.go"]

## (6) Add the statically linked binary to a distroless image
FROM golang:1.17 as production-small

RUN rm -rf /go
RUN rm -rf /app
RUN rm -rf /usr/local/go
COPY --from=production /app/main /bin/main

CMD ["/bin/main"]

FROM golang:1.17 as build-cli-env

RUN git clone https://github.com/onflow/flow-cli.git /flow-cli
WORKDIR /flow-cli

# FIX: Let's not gamble and stick to v0.34.0. Backward compatibility can be checked this way.
RUN git checkout 6c240a76ec2bb5d5685afeb0898eed0ea1bd0059
RUN go mod download
# FIX: make sure no further steps update modules later, so that we can debug regressions
RUN go mod vendor

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

FROM golang:1.17 as flow-cli

RUN rm -rf /go
RUN rm -rf /app
RUN rm -rf /usr/local/go
COPY --from=build-cli /flow-cli/main /bin/flow

CMD ["/bin/bash"]

FROM flow-cli as flow-e2e-test

COPY ./flow-localnet.json /root/flow-localnet.json
WORKDIR /root
CMD flow -f /root/flow-localnet.json -n flow_api blocks get latest