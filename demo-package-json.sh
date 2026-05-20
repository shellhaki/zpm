#!/bin/bash
# Demo script for ZPM package.json support

# Uhh i dont have energy for comments soo....
set -e

echo "=== ZPM package.json Support Demo ==="
echo

REPO_ROOT="/home/haki/zpm"
ZPM="$REPO_ROOT/zig-out/bin/zpm"


if [ ! -f "$ZPM" ]; then
    echo "Error: zpm binary not found at $ZPM"
    exit 1
fi

=
echo "Cleaning up any existing demo processes..."
$ZPM purge demo-prod 2>/dev/null || true
$ZPM purge demo-dev 2>/dev/null || true
echo


echo "Test 1: Run default 'start' script"
echo "Command: cd $REPO_ROOT/demo-app && $ZPM start --name demo-prod"
(cd "$REPO_ROOT/demo-app" && $ZPM start --name demo-prod)
echo

echo "Test 2: Run specific 'dev' script"
echo "Command: cd $REPO_ROOT/demo-app && $ZPM start --name demo-dev --script dev"
(cd "$REPO_ROOT/demo-app" && $ZPM start --name demo-dev --script dev)
echo


echo "Test 3: List all processes"
$ZPM list
echo


echo "Test 4: Checking registry (zpm.json)..."
if [ -f "$REPO_ROOT/demo-app/zpm.json" ]; then
    echo "Registry contents:"
    cat "$REPO_ROOT/demo-app/zpm.json"
fi
echo


echo "Test 5: Stopping processes"
$ZPM stop demo-prod
$ZPM stop demo-dev
echo

echo "Test 6: Purging processes"
$ZPM purge demo-prod
$ZPM purge demo-dev
echo

echo "✓ Demo complete! ZPM now supports package.json scripts."
