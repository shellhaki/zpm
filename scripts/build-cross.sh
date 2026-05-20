#!/usr/bin/env bash
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUTPUT_DIR="${OUTPUT_DIR:-$PROJECT_ROOT/releases}"
BUILD_DIR="${BUILD_DIR:-$PROJECT_ROOT/build-cross}"
OPTIMIZE="${OPTIMIZE:-ReleaseSafe}"

targets=(
  "linux-x86_64:x86_64-linux"
  "linux-aarch64:aarch64-linux"
  "macos-x86_64:x86_64-macos"
  "macos-aarch64:aarch64-macos"
)

mkdir -p "$OUTPUT_DIR" "$BUILD_DIR"

echo "Building ZPM release archives"
echo "Output: $OUTPUT_DIR"

for target in "${targets[@]}"; do
  name="${target%%:*}"
  zig_target="${target#*:}"
  prefix="$BUILD_DIR/$name"
  archive="$OUTPUT_DIR/zpm-$name.tar.gz"

  echo ""
  echo "Building $name ($zig_target)"

  rm -rf "$prefix"
  zig build -Dtarget="$zig_target" -Doptimize="$OPTIMIZE" --prefix "$prefix"

  tar -czf "$archive" -C "$prefix/bin" zpm zpmd
  echo "Wrote $archive"
done

echo ""
echo "Done. Release archives:"
ls -lh "$OUTPUT_DIR"/zpm-*.tar.gz
