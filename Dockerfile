FROM golang:1.24-alpine3.20 AS build-env

SHELL ["/bin/sh", "-ecuxo", "pipefail"]

RUN set -eux; apk add --no-cache \
    ca-certificates \
    build-base \
    git \
    linux-headers \
    bash \
    binutils-gold \
    just

WORKDIR /code

ADD go.mod go.sum ./
RUN go mod download

# force it to use static lib (from above) not standard libgo_cosmwasm.so file
# then log output of file /code/bin/shinzohubd
# then ensure static linking
RUN LEDGER_ENABLED=false BUILD_TAGS=muslc LINK_STATICALLY=true make build \
  && file /code/build/shinzohubd \
  && echo "Ensuring binary is statically linked ..." \
  && (file /code/build/shinzohubd | grep "statically linked")

# --------------------------------------------------------
FROM alpine:3.21

COPY --from=build-env /code/build/shinzohubd /usr/bin/shinzohubd

RUN apk add --no-cache ca-certificates curl make bash jq sed

WORKDIR /opt

# rest server, tendermint p2p, tendermint rpc
EXPOSE 1317 26656 26657 8545 8546

CMD ["/usr/bin/shinzohubd", "version"]
