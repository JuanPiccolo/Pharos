#!/bin/bash

# 1. Anchor the script to its own directory
DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$DIR"

# 2. Setup directory structure
mkdir -p BridgeBuild

# 3. Handle Go Module
if [ ! -f "go.mod" ]; then
    echo "Initializing Go Module..."
    go mod init cgo-bridge
fi

# 4. Build Go Engine
# We output to BridgeBuild/libengine.a
echo "Step 1: Building Go Static Archive..."
go build -buildmode=c-archive -o BridgeBuild/libengine.a .

# 5. Compile the Final App
# CRITICAL: The order must be Source -> Library -> Pkg-Config
echo "Step 2: Linking C Bridge..."
gcc BridgeBuild/bridgebuild.c \
    -o my_app \
    -I"$DIR/BridgeBuild" \
    -L"$DIR/BridgeBuild" \
    -lengine \
    $(pkg-config --cflags --libs gtk+-3.0 webkit2gtk-4.1) \
    -lpthread -ldl -lm

echo "------------------------------------"
# Check the specific silo folder for the executable
if [ -f "BridgeBuild/my_app" ]; then
    echo "SUCCESS: BridgeBuild/my_app created."
else
    echo "FAILED: Check GCC error output above."
fi