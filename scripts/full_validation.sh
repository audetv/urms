#!/bin/bash
# full_validation.sh - Complete validation suite for URMS-OS

set -e  # Exit on any error

echo "🚀 URMS-OS Full Validation Suite"
echo "================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print status
print_status() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✅ $2${NC}"
    else
        echo -e "${RED}❌ $2${NC}"
        exit 1
    fi
}

# Navigate to backend directory
cd "$(dirname "$0")/../backend" || exit 1

echo ""
echo "📋 Phase 1: Architecture Validation"
echo "-----------------------------------"

# Run architecture audit
../scripts/architecture_audit.sh
ARCHITECTURE_RESULT=$?
print_status $ARCHITECTURE_RESULT "Architecture compliance"

echo ""
echo "🧪 Phase 2: Unit Tests"
echo "----------------------"

# Run core domain tests
echo "🔍 Testing domain layer..."
go test ./internal/core/domain/ -v -count=1
DOMAIN_TEST_RESULT=$?
print_status $DOMAIN_TEST_RESULT "Domain layer tests"

# Run core services tests  
echo "🔍 Testing services layer..."
go test ./internal/core/services/ -v -count=1
SERVICES_TEST_RESULT=$?
print_status $SERVICES_TEST_RESULT "Services layer tests"

echo ""
echo "🔧 Phase 3: Compilation Checks"
echo "------------------------------"

# Check core compilation
echo "🔍 Compiling core layer..."
go build ./internal/core/...
CORE_BUILD_RESULT=$?
print_status $CORE_BUILD_RESULT "Core layer compilation"

# Check infrastructure compilation
echo "🔍 Compiling infrastructure layer..."
go build ./internal/infrastructure/...
INFRASTRUCTURE_BUILD_RESULT=$?
print_status $INFRASTRUCTURE_BUILD_RESULT "Infrastructure layer compilation"

# Check test application compilation
echo "🔍 Compiling test applications..."
go build ./cmd/test-new-architecture/...
TEST_APP_RESULT=$?
print_status $TEST_APP_RESULT "Test application compilation"

echo ""
echo "📊 Phase 4: Code Quality"
echo "-----------------------"

# Check for gofmt issues
echo "🔍 Checking code formatting..."
if [ -n "$(gofmt -l .)" ]; then
    echo -e "${YELLOW}⚠️  Code formatting issues found:${NC}"
    gofmt -l .
    FORMAT_RESULT=1
else
    FORMAT_RESULT=0
fi
print_status $FORMAT_RESULT "Code formatting"

# Simple dependency check
echo "🔍 Checking for obvious issues..."
if go list -f '{{.Deps}}' ./internal/core/domain/... | grep -q "github.com"; then
    echo -e "${RED}❌ Domain layer has external dependencies${NC}"
    DEPENDENCY_RESULT=1
else
    DEPENDENCY_RESULT=0
fi
print_status $DEPENDENCY_RESULT "Domain layer purity"

echo ""
echo "🎯 Final Results"
echo "---------------"

echo -e "${GREEN}✅ All validations completed successfully!${NC}"
echo ""
echo "📈 Summary:"
echo "  • Architecture: ✅ Compliant"
echo "  • Unit Tests: ✅ Passing" 
echo "  • Compilation: ✅ Successful"
echo "  • Code Quality: ✅ Good"
echo ""
echo "🚀 Project is ready for Phase 1B development!"