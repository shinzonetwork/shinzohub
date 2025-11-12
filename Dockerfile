FROM golang:1.24.7 AS builder

WORKDIR /app

# Cache deps
COPY go.* /app/
RUN go mod download

# Build
COPY . /app
RUN --mount=type=cache,target=/root/.cache scripts/build.sh

# Deployment entrypoint
FROM debian:trixie-slim

COPY docker/entrypoint.sh /usr/local/bin/entrypoint.sh
#COPY docker/config.toml /etc/shinzohub/config.toml
#COPY docker/app.toml /etc/shinzohub/app.toml
COPY --from=builder /app/build/shinzohubd /usr/local/bin/shinzohubd

RUN useradd --create-home --home-dir /home/node node && mkdir /shinzohub && chown node:node /shinzohub && ln -s /shinzohub /home/node/.shinzohub && chown node:node -R /home/node

# MONIKER sets the node moniker
ENV MONIKER="node"
# CHAIN_ID sets the id for the chain which will be initialized
ENV CHAIN_ID="shinzohub-dev"

# GENESIS_PATH is an optional variable which if set must point to a genesis file mounted in the container.
# The file is copied to the configuration directory during the first container initialization
# If empty, the entrypoint will generate a new genesis
ENV GENESIS_PATH=""

# MNEMONIC_PATH is an optional varible which, if set, must point to a file containing a 
# cosmos key mnemonic. The mnemonic will be used to restore the node operator / validator key.
# If empty, the entrypoint will generate a new key
ENV MNEMONIC_PATH=""

# CONSENSUS_KEY_PATH is an optional variable which, if set, must point to a file containg
# a comebft consesus key for the validator.
# If empty, the entrypoint will generate a new key
ENV CONSENSUS_KEY_PATH=""

# COMET_NODE_KEY_PATH is an optional variable which, if set, must point to a file containg
# a comebft p2p node key.
# If empty, the entrypoint will generate a new key
ENV COMET_NODE_KEY_PATH=""

# COMET_CONFIG_PATH is an optional variable which, if set, will overwrite
# the default cofig.toml with the provided file.
ENV COMET_CONFIG_PATH=""

# APP_CONFIG_PATH is an optional variable which, if set, will overwrite
# the default app.toml with the provided file.
ENV APP_CONFIG_PATH=""

ENV STANDALONE=""

# Comet P2P Port
EXPOSE 26656

# Comet RPC Port
EXPOSE 26657

# shinzohub GRPC Port
EXPOSE 9090

# shinzohub HTTP API Port
EXPOSE 1317

USER node
VOLUME ["/shinzohub"]
ENTRYPOINT ["entrypoint.sh"]
CMD ["shinzohubd", "start"]