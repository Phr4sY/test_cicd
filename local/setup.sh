#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
KEYS_DIR="$SCRIPT_DIR/keys"
WEB_KEYS="$KEYS_DIR/web"
WORKER_KEYS="$KEYS_DIR/worker"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'

info()  { echo -e "${GREEN}[INFO]${NC} $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
die()   { echo -e "${RED}[ERROR]${NC} $*" >&2; exit 1; }

# ── Start the Podman machine if needed ────────────────────────────────────────
#
# The machine was created by Podman Desktop (5.7.1) which uses krunkit.
# Homebrew Podman 6.0.0 generates an incompatible --timesync flag for krunkit,
# so we must use /opt/podman/bin/podman (5.7.1) to start/stop the VM.
# Both versions share the same socket once the machine is running.
#
PODMAN_MACHINE_BIN="${PODMAN_MACHINE_BIN:-/opt/podman/bin/podman}"
MACHINE_NAME="${MACHINE_NAME:-podman-machine}"

if ! podman ps >/dev/null 2>&1; then
    if [[ ! -x "$PODMAN_MACHINE_BIN" ]]; then
        die "Cannot reach Podman socket and $PODMAN_MACHINE_BIN not found.\nStart your Podman machine manually: podman machine start"
    fi
    info "Starting Podman machine '$MACHINE_NAME' via $PODMAN_MACHINE_BIN …"
    "$PODMAN_MACHINE_BIN" machine start "$MACHINE_NAME" &
    until podman ps >/dev/null 2>&1; do sleep 2; done
    info "Machine is up."
fi

# ── Generate keys ─────────────────────────────────────────────────────────────

if [[ -f "$WEB_KEYS/session_signing_key" ]]; then
    warn "Keys already exist — skipping generation. Delete local/keys/ to regenerate."
else
    info "Generating Concourse keys…"

    mkdir -p "$WEB_KEYS" "$WORKER_KEYS"

    openssl genrsa -out "$WEB_KEYS/session_signing_key" 2048 2>/dev/null
    openssl rsa -in "$WEB_KEYS/session_signing_key" \
                -pubout -out "$WEB_KEYS/session_signing_key.pub" 2>/dev/null

    # -m PEM forces PKCS#1 format — Concourse 8 rejects the default PKCS#8
    ssh-keygen -t rsa -b 2048 -f "$WEB_KEYS/tsa_host_key"     -N "" -m PEM -q
    ssh-keygen -t rsa -b 2048 -f "$WORKER_KEYS/worker_key"     -N "" -m PEM -q

    cp "$WORKER_KEYS/worker_key.pub"  "$WEB_KEYS/authorized_worker_keys"
    cp "$WEB_KEYS/tsa_host_key.pub"   "$WORKER_KEYS/tsa_host_key.pub"

    info "Keys generated."
fi

# ── Start stack ───────────────────────────────────────────────────────────────

info "Starting Concourse stack (pulling images on first run)…"
cd "$SCRIPT_DIR"
podman compose up -d

# ── Wait for web to be ready ──────────────────────────────────────────────────

info "Waiting for Concourse web to be ready…"
for i in $(seq 1 60); do
    if curl -sf http://localhost:8080/api/v1/info >/dev/null 2>&1; then
        echo ""
        break
    fi
    printf "."
    sleep 3
done

if ! curl -sf http://localhost:8080/api/v1/info >/dev/null 2>&1; then
    die "Concourse web did not come up. Check logs:\n  podman compose -f $SCRIPT_DIR/compose.yml logs web"
fi

# ── Log in with fly ───────────────────────────────────────────────────────────

# Prefer a local fly8 if it exists (needed because Concourse 8 ships arm64
# images but the bundled fly on macOS may be 7.x which refuses to talk to 8.x)
if [[ -x "$HOME/bin/fly8" ]]; then
    FLY="$HOME/bin/fly8"
elif command -v fly &>/dev/null; then
    FLY=fly
else
    die "fly CLI not found. Install from http://localhost:8080 after the stack is up."
fi

info "Logging in with fly ($($FLY --version))…"
$FLY -t local login \
    --concourse-url http://localhost:8080 \
    --username admin \
    --password admin

# Download the matching fly if there is a version mismatch
if $FLY -t local pipelines >/dev/null 2>&1; then
    : # version is compatible
else
    info "fly version mismatch — downloading matching fly from the server…"
    mkdir -p "$HOME/bin"
    curl -sfL "http://localhost:8080/api/v1/cli?arch=arm64&platform=darwin" \
         -o "$HOME/bin/fly8" && chmod +x "$HOME/bin/fly8"
    FLY="$HOME/bin/fly8"
    $FLY -t local login \
        --concourse-url http://localhost:8080 \
        --username admin \
        --password admin
fi

info "Setting the pipeline…"
$FLY -t local set-pipeline -p test-cicd \
    -c "$SCRIPT_DIR/../pipeline.yml" \
    -v deploy_url=http://placeholder.example.com \
    -v deploy_token=changeme \
    -v app_version=1.0.0 \
    --non-interactive

$FLY -t local unpause-pipeline -p test-cicd

echo ""
echo -e "${GREEN}✓ Concourse is up at http://localhost:8080${NC}"
echo "  Username: admin  |  Password: admin"
echo "  Target:   local"
echo ""
echo "Useful fly commands:"
echo "  fly -t local pipelines"
echo "  fly -t local trigger-job -j test-cicd/validate-endpoint --watch"
echo "  fly -t local trigger-job -j test-cicd/deploy-app --watch"
echo ""
echo "To stop:  ./teardown.sh"
