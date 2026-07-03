#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

cd "$SCRIPT_DIR"

echo "Stopping Concourse containers (data volumes are preserved)…"
podman compose down

echo "Done. Run ./setup.sh to start again."
echo ""
echo "To also delete data volumes (wipes Postgres and worker state):"
echo "  podman compose down -v"
echo ""
echo "(The Podman machine is left running — stop it manually with:"
echo "  /opt/podman/bin/podman machine stop podman-machine)"
