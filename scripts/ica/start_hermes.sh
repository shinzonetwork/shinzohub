HERMES_HOME="$HOME/.hermes"

SOURCEHUB_CHAIN_ID="sourcehub"
SHINZOHUB_CHAIN_ID="9001"

rm -rf $HERMES_HOME

# Make hermes dir and copy the config
mkdir $HERMES_HOME
mkdir $HERMES_HOME/bin/
tar -C $HERMES_HOME/bin/ -vxzf $HOME/hermes-v1.13.3-aarch64-apple-darwin.tar.gz
cp ./hermes_config.toml "$HERMES_HOME/config.toml"

xattr -d com.apple.quarantine "$HERMES_HOME/bin/hermes" || true

# Add hermes key (same for both chains)
echo "divert tenant reveal hire thing jar carry lonely magic oak audit fiber earth catalog cheap merry print clown portion speak daring giant weird slight" > /tmp/mnemonic.txt
hermes keys add --chain $SOURCEHUB_CHAIN_ID --mnemonic-file /tmp/mnemonic.txt
hermes keys add --chain $SHINZOHUB_CHAIN_ID --mnemonic-file /tmp/mnemonic.txt --hd-path "m/44'/60'/0'/0/0"
rm /tmp/mnemonic.txt

hermes keys list --chain $SHINZOHUB_CHAIN_ID
hermes keys list --chain $SOURCEHUB_CHAIN_ID

echo "==> Creating IBC connection..."
hermes create connection \
  --a-chain $SOURCEHUB_CHAIN_ID \
  --b-chain $SHINZOHUB_CHAIN_ID

sleep 1

echo "==> Starting Hermes..."
hermes start