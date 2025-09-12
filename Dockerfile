FROM golang:1.23.6-alpine3.20 AS build-env

SHELL ["/bin/sh", "-ecuxo", "pipefail"]

RUN set -eux; apk add --no-cache \
    ca-certificates \
    build-base \
    git \
    linux-headers \
    bash \
    binutils-gold

WORKDIR /code

ADD go.mod go.sum ./
RUN go mod download

# Copy over code
COPY . /code

# --------------------------------------------------------
FROM alpine:3.21

COPY --from=build-env /code/build/shinzohubd /usr/bin/shinzohubd

RUN apk add --no-cache ca-certificates curl make bash jq sed

WORKDIR /opt

# rest server, tendermint p2p, tendermint rpc
EXPOSE 1317 26656 26657 8545 8546

CMD ["/usr/bin/shinzohubd", "version"]
