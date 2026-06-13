#!/usr/bin/env bash
set -euo pipefail

# Defaults — override via environment variables.
IMAGE="${IPTV_IMAGE:-iptv-builder:latest}"
CONFIG_DIR="${IPTV_CONFIG:-$(pwd)/config}"
OUTPUT_DIR="${IPTV_OUTPUT:-$(pwd)/output}"
CACHE_DIR="${IPTV_CACHE:-$(pwd)/cache}"

# Ensure output and cache directories exist so Docker doesn't create
# them as root-owned.
mkdir -p "${OUTPUT_DIR}" "${CACHE_DIR}"

echo "=== IPTV Builder ==="
echo "Image:      ${IMAGE}"
echo "Config dir: ${CONFIG_DIR}"
echo "Output dir: ${OUTPUT_DIR}"
echo "Cache dir:  ${CACHE_DIR}"
echo ""

docker run --rm \
  --name iptv-builder \
  -v "${CONFIG_DIR}:/config:ro" \
  -v "${OUTPUT_DIR}:/output" \
  -v "${CACHE_DIR}:/cache" \
  "${IMAGE}"

echo ""
echo "Done. Output: ${OUTPUT_DIR}/final.m3u"
