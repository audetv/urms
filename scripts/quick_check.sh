#!/bin/bash
# quick_check.sh - Fast validation for development

set -e

echo "⚡ URMS-OS Quick Check"
cd "$(dirname "$0")/../backend" || exit 1

# Run tests
echo "🧪 Running tests..."
go test ./internal/core/... -v

# Build check
echo "🔧 Checking compilation..."
go build ./internal/...

# Architecture check
echo "📐 Architecture validation..."
../scripts/architecture_audit.sh

echo "✅ Quick check completed!"