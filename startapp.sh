#!/bin/sh

# 1. Get the absolute path to the folder containing THIS script
HERE=$(dirname "$(realpath "$0")")

# 2. "CD" into that folder so the binary's relative paths work
cd "$HERE"

# 3. Launch the binary
# Use ./ to ensure it hits the local file, and pass any arguments along
./my_app "$@"
