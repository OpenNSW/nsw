#!/bin/sh
set -eu

: "${TEMPORAL_ADDRESS:?TEMPORAL_ADDRESS is required}"
: "${DEFAULT_NAMESPACE:?DEFAULT_NAMESPACE is required}"

RETENTION_DAYS="${TEMPORAL_NAMESPACE_RETENTION_DAYS:-3}"

echo "[create-namespace] Ensuring namespace '${DEFAULT_NAMESPACE}' exists on ${TEMPORAL_ADDRESS}..."

if tctl --address "${TEMPORAL_ADDRESS}" namespace describe --namespace "${DEFAULT_NAMESPACE}" >/dev/null 2>&1; then
	echo "[create-namespace] Namespace already exists."
	exit 0
fi

# tctl retention is specified in days.
# If this is a race (multiple attempts), we tolerate it.
tctl --address "${TEMPORAL_ADDRESS}" namespace register \
	--namespace "${DEFAULT_NAMESPACE}" \
	--retention "${RETENTION_DAYS}" || true

if tctl --address "${TEMPORAL_ADDRESS}" namespace describe --namespace "${DEFAULT_NAMESPACE}" >/dev/null 2>&1; then
	echo "[create-namespace] Namespace created."
	exit 0
fi

echo "[create-namespace] Failed to create/verify namespace." >&2
exit 1


