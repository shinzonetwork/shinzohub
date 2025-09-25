#!/bin/bash

set -e

POLICY_FILE="acp/shinzo.yaml"
POLICY_NAME="shinzo"
POLICY_ID_FILE=".shinzohub/policy_id"
KEY_NAME="validator" 

# Helper function to find a policy ID by name
find_policy_id_by_name() {
  local name="$1"
  local debug_file=".shinzohub/policy_debug.jsonl"
  mkdir -p .shinzohub
  : > "$debug_file"  # Truncate debug file at start of each call
  RAW_POLICY_IDS=$(sourcehubd query acp policy-ids --chain-id=sourcehub-dev --output json)
  POLICY_IDS=$(echo "$RAW_POLICY_IDS" | jq -r '(.policy_ids // .ids // []) | .[]')
  if [[ "$POLICY_IDS" == "" ]]; then
    echo ""
    return
  fi
  for ID in $POLICY_IDS; do
    POLICY_JSON=$(sourcehubd query acp policy "$ID" --chain-id=sourcehub-dev --output json)
    echo "$POLICY_JSON" >> "$debug_file"
    POLICY_NAME_FIELD=$(echo "$POLICY_JSON" | jq -r '.record.policy.name // .policy.name // .name')
    if [[ "$POLICY_NAME_FIELD" == "$name" ]]; then
      echo "$ID"
      return
    fi
  done
  echo ""
}

# Check if the policy already exists
POLICY_ID=$(find_policy_id_by_name "$POLICY_NAME")
if [[ -n "$POLICY_ID" ]]; then
  echo "Policy with name '$POLICY_NAME' already exists with ID: $POLICY_ID"
  exit 0
fi

# Upload the policy
echo "Uploading policy from $POLICY_FILE..."
sourcehubd tx acp create-policy "$POLICY_FILE" \
  --from "$KEY_NAME" \
  --chain-id=sourcehub-dev \
  --keyring-backend=test \
  --gas=auto \
  --fees 500uopen \
  -y
echo "Transaction submitted, waiting for confirmation..."
sleep 5

# Re-query to get the new policy ID
POLICY_ID=$(find_policy_id_by_name "$POLICY_NAME")
if [[ -z "$POLICY_ID" ]]; then
  echo "ERROR: Policy with name '$POLICY_NAME' not found after upload!"
  exit 1
fi

echo "Found policy ID: $POLICY_ID"

# Save the policy ID to a file
mkdir -p .shinzohub
echo -n "$POLICY_ID" > "$POLICY_ID_FILE"
echo "Saved policy ID to $POLICY_ID_FILE"

echo "Policy setup complete." 