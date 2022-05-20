# NOTE: Must be run in the context of the repo's root directory

## (1) Download a suitable version of Go
FROM golang:1.18 AS build-setup

# Add optional items like apt install -y make cmake gcc g++

## (2) Build the app binary
FROM build-setup AS build-env

COPY src /app/src

WORKDIR /app

RUN go build -o /app src/main/api-service.go
RUN ls /app

CMD ["go", "run", "src/main/api-service.go"]

## (3) Test the project
FROM build-setup AS test

COPY src /app/src

WORKDIR /app/src

CMD ["go", "test", "./..."]


## (4) Add the statically linked binary to a distroless image
FROM scratch as production

COPY --from=build-env /app/api-service /bin/app

CMD ["/bin/app"]