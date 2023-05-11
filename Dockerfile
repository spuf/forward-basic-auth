ARG go_version=1.20
ARG alpine_version=3.17

###
FROM golang:${go_version}-alpine${alpine_version} AS base

# golangci deps
RUN apk add --no-cache git build-base

ARG golangci_version=1.52.2
RUN wget -O- -nv 'https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh' | sh -s "v${golangci_version}"

WORKDIR /go/src/forward-basic-auth

COPY go.mod go.sum ./
RUN go mod download
RUN go mod verify

COPY . ./

RUN golangci-lint run ./... --timeout=10m

###
FROM base as build

ARG version=""
RUN go build -ldflags="-X main.Version=${version}" -o /go/bin/forward-basic-auth

###
FROM alpine:${alpine_version}
RUN mkdir -m 0777 -p /mnt/db && chown nobody:nogroup /mnt/db
COPY --from=build /go/bin/forward-basic-auth /forward-basic-auth

STOPSIGNAL SIGINT
USER nobody:nogroup
ENTRYPOINT ["/forward-basic-auth"]
