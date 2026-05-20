#!/usr/bin/env sh
set -eu

REPO="${ZPM_REPO:-shellhaki/zpm}"
TAG="${ZPM_TAG:-latest}"
INSTALL_DIR="${ZPM_INSTALL_DIR:-$HOME/.local/bin}"

need() {
    if ! command -v "$1" >/dev/null 2>&1; then
        echo "error: '$1' is required" >&2
        exit 1
    fi
}

detect_target() {
    os="$(uname -s)"
    arch="$(uname -m)"

    case "$arch" in
        x86_64|amd64) arch="x86_64" ;;
        arm64|aarch64) arch="aarch64" ;;
        *)
            echo "error: unsupported architecture: $arch" >&2
            exit 1
            ;;
    esac

    case "$os" in
        Linux) echo "linux-$arch" ;;
        Darwin) echo "macos-$arch" ;;
        *)
            echo "error: unsupported operating system: $os" >&2
            echo "ZPM releases currently support Linux and macOS." >&2
            exit 1
            ;;
    esac
}

need curl
need install
need mktemp
need tar
need uname

target="$(detect_target)"
asset="zpm-$target.tar.gz"

if [ "$TAG" = "latest" ]; then
    url="https://github.com/$REPO/releases/latest/download/$asset"
else
    url="https://github.com/$REPO/releases/download/$TAG/$asset"
fi

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT INT TERM

echo "Installing ZPM for $target"
echo "Downloading $url"

curl -fL "$url" -o "$tmpdir/$asset"
tar -xzf "$tmpdir/$asset" -C "$tmpdir"

mkdir -p "$INSTALL_DIR"
install -m 0755 "$tmpdir/zpm" "$INSTALL_DIR/zpm"

if [ -f "$tmpdir/zpmd" ]; then
    install -m 0755 "$tmpdir/zpmd" "$INSTALL_DIR/zpmd"
fi

echo "Installed zpm to $INSTALL_DIR/zpm"

case ":$PATH:" in
    *":$INSTALL_DIR:"*) ;;
    *)
        echo ""
        echo "Add this to your shell profile if zpm is not found:"
        echo "  export PATH=\"$INSTALL_DIR:\$PATH\""
        ;;
esac

echo ""
"$INSTALL_DIR/zpm" 2>/dev/null || true
