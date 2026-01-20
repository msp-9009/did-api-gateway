#!/bin/bash
# Quick start script for DID:Web test server

set -e

echo "🌐 Building DID:Web Test Server..."
cd "$(dirname "$0")"

# Build the server
go build -o did-web-test-server main.go

echo "✅ Build complete!"
echo ""
echo "Starting server on port 8888..."
echo "DID will be: did:web:localhost:8888"
echo ""
echo "Press Ctrl+C to stop the server"
echo "Open http://localhost:8888 in your browser for instructions"
echo ""

# Run the server
./did-web-test-server -port 8888 -domain "localhost:8888"
