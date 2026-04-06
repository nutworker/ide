#!/bin/bash

# Installation script for ide

set -e

echo "Building ide..."
go build -o ide

echo ""
echo "IDE built successfully!"
echo ""
echo "To install system-wide (requires sudo):"
echo "  sudo cp ide /usr/local/bin/"
echo ""
echo "To install for current user:"
echo "  mkdir -p ~/bin && cp ide ~/bin/"
echo "  (Make sure ~/bin is in your PATH)"
echo ""
echo "To run from current directory:"
echo "  ./ide"
echo ""
