#!/bin/bash
# install.sh - Prepares the file service environment

set -e

echo "🔧 Setting up File VM"

# Create files directory if it doesn't exist
mkdir -p files

# Download a sample image if not already present
if [ ! -f files/sample.jpg ]; then
    echo "📥 Downloading sample image..."
    curl -o files/sample.jpg https://picsum.photos/800/600 || \
    wget -O files/sample.jpg https://picsum.photos/800/600 || \
    echo "⚠️ Could not download sample image. Please place your own file in files/"
fi

# Initialize go module if needed
if [ ! -f go.mod ]; then
    go mod init file
fi

# Tidy modules (none, but keeps things clean)
go mod tidy

echo "✅ Setup complete. Run './run.sh' to start the server."