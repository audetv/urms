#!/bin/bash
# validate_architecture.sh

echo "🔍 Checking architecture compliance..."

# Переходим в директорию backend где находится go.mod
cd "$(dirname "$0")/../backend" || exit 1

# Check core/ has no infrastructure imports
echo "1. Checking core/ dependencies..."
if go list -f '{{.Deps}}' ./internal/core/... | grep -q "infrastructure"; then
    echo "❌ FAIL: core/ imports infrastructure/"
    exit 1
fi

# Check interfaces are defined
echo "2. Checking interface definitions..."
if ! find internal/core/ports -name "*.go" -type f | grep -q ".go"; then
    echo "❌ FAIL: No interfaces in core/ports/"
    exit 1
fi

# Check domain has no external dependencies
echo "3. Checking domain layer purity..."
if go list -f '{{.Deps}}' ./internal/core/domain/... | grep -q "github.com"; then
    echo "❌ FAIL: domain/ has external dependencies"
    exit 1
fi

# Check infrastructure implements ports
echo "4. Checking infrastructure implements ports..."
if ! go build ./internal/infrastructure/... > /dev/null 2>&1; then
    echo "❌ FAIL: Infrastructure compilation failed"
    exit 1
fi

echo "✅ All architecture checks passed"