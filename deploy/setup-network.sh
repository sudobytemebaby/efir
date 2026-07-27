#!/bin/bash
set -euo pipefail

NETWORK_NAME="efir-network"
SUBNET="172.20.0.0/24"
GATEWAY="172.20.0.1"

echo "Setting up Podman network: ${NETWORK_NAME}"

if podman network exists "${NETWORK_NAME}" 2>/dev/null; then
  echo "Network '${NETWORK_NAME}' already exists, removing..."
  podman network rm "${NETWORK_NAME}"
fi

podman network create \
  --driver bridge \
  --subnet "${SUBNET}" \
  --gateway "${GATEWAY}" \
  "${NETWORK_NAME}"

echo "Network '${NETWORK_NAME}' created successfully"
