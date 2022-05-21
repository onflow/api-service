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
WORKDIR /app/src
COPY src/go.mod /app/src
COPY src/go.sum /app/src
RUN go mod download

# FIX: This generates code marked by `go:build relic` and `+build relic`. See `combined_verifier_v3.go`.
RUN go mod download github.com/onflow/flow-go/crypto@v0.24.3
RUN cd $GOPATH/pkg/mod/github.com/onflow/flow-go/crypto@v0.24.3 && go generate && go build

## (3) Build the app binary
FROM build-dependencies AS build-env

COPY src /app/src
WORKDIR /app/src

# Fix Devs should review all what they use to limit build time
RUN cat go.sum

# FIX: Without -tags=relic we get undefined: "github.com/onflow/flow-go/consensus/hotstuff/verification".NewCombinedVerifier
RUN go build -v -tags=relic -o /app main/api-service.go

CMD ["go", "run", "main/api-service.go"]

## (5) Add the statically linked binary to a distroless image
FROM scratch as production

COPY --from=build-env /app/api-service /bin/app

CMD ["/bin/app"]