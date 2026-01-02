#!/bin/bash
set -e

export PATH=$HOME/go/bin:$PATH

# Function to check and install a tool
ensure_tool() {
    local tool_name=$1
    local install_cmd=$2

    if ! command -v "$tool_name" &> /dev/null; then
        echo "$tool_name not found. Installing..."
        eval "$install_cmd"
    fi
}

# Ensure goimports
ensure_tool "goimports" "go install golang.org/x/tools/cmd/goimports@latest"

# Ensure golangci-lint
ensure_tool "golangci-lint" "go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
