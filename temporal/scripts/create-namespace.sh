#!/bin/sh
set -eu

: "${TEMPORAL_ADDRESS:?TEMPORAL_ADDRESS is required}"
: "${DEFAULT_NAMESPACE:?DEFAULT_NAMESPACE is required}"

RETENTION_DAYS="${TEMPORAL_NAMESPACE_RETENTION_DAYS:-3}"

echo "[create-namespace] Ensuring namespace '${DEFAULT_NAMESPACE}' exists on ${TEMPORAL_ADDRESS}..."

if temporal operator namespace describe "${DEFAULT_NAMESPACE}" \
     --address "${TEMPORAL_ADDRESS}" >/dev/null 2>&1; then
  echo "[create-namespace] Namespace already exists."
  exit 0
fi

temporal operator namespace create \
  --address "${TEMPORAL_ADDRESS}" \
  --retention "${RETENTION_DAYS}d" \
  "${DEFAULT_NAMESPACE}" || true

if temporal operator namespace describe "${DEFAULT_NAMESPACE}" \
     --address "${TEMPORAL_ADDRESS}" >/dev/null 2>&1; then
  echo "[create-namespace] Namespace created."
  exit 0
fi

echo "[create-namespace] Failed to create/verify namespace." >&2
exit 1