#!/usr/bin/env bash
set -euo pipefail

CONFIG=/home/hermes/.hermes/config.toml
MNEMONIC="divert tenant reveal hire thing jar carry lonely magic oak audit fiber earth catalog cheap merry print clown portion speak daring giant weird slight"

# Stage the config in a directory the `hermes` user owns.
mkdir -p /home/hermes/.hermes
cp /tmp/hermes-config.toml "$CONFIG"

echo "==> Adding hermes keys..."
echo "$MNEMONIC" > /tmp/mnemonic.txt
hermes --config "$CONFIG" keys add --chain sourcehub --mnemonic-file /tmp/mnemonic.txt
hermes --config "$CONFIG" keys add --chain 91273002 --mnemonic-file /tmp/mnemonic.txt --hd-path "m/44'/60'/0'/0/0"
rm /tmp/mnemonic.txt

echo "==> Creating IBC connection (retrying until both chains are ready)..."
for i in $(seq 1 60); do
  if hermes --config "$CONFIG" create connection \
       --a-chain sourcehub --b-chain 91273002; then
    echo "==> Connection created."
    break
  fi
  if [ "$i" -eq 60 ]; then
    echo "ERROR: hermes create connection failed after 60 attempts"
    exit 1
  fi
  echo "    attempt $i failed, retrying in 2s..."
  sleep 2
done

echo "==> Starting hermes..."
exec hermes --config "$CONFIG" start
