
# URMS-OS Architecture Validation Rules
**Automated Checks & Manual Review Guidelines**
**Version: 1.0**

## 🔧 Automated Validation Scripts

### 1. Dependency Check Script
```bash
#!/bin/bash
# validate_architecture.sh

echo "🔍 Checking architecture compliance..."

# Check core/ has no infrastructure imports
echo "1. Checking core/ dependencies..."
if go list -f '{{.Deps}}' ./core/... | grep -q "infrastructure"; then
    echo "❌ FAIL: core/ imports infrastructure/"
    exit 1
fi

# Check interfaces are defined
echo "2. Checking interface definitions..."
if ! find core/ports -name "*.go" -type f | grep -q ".go"; then
    echo "❌ FAIL: No interfaces in core/ports/"
    exit 1
fi

echo "✅ All architecture checks