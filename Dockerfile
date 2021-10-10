ARG go_version=1.17
ARG alpine_version=3.14

###
FROM golang:${go_version}-alpine${alpine_version} AS base

# golangci deps
RUN apk add --no-cache curl git build-base

ARG golangci_version=1.42.1
RUN curl -sSfL 'https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh' | sh -s "v${golangci_version}"

WORKDIR /go/src/forward-basic-auth

COPY go.mod go.sum ./
RUN go mod download
RUN go mod verify

COPY . ./
RUN goa gen github.com/spuf/forward-basic-auth/design

RUN golangci-lint run ./...

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
