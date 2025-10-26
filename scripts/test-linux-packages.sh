#!/bin/bash
# CloudWorkstation Linux Package Testing Script
# Tests RPM and DEB packages in Docker containers

set -euo pipefail

# Colors and formatting
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color
BOLD='\033[1m'

# Configuration
PACKAGE_NAME="prism"
VERSION="${VERSION:-0.4.2}"
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEST_RESULTS_DIR="$PROJECT_ROOT/test_results/packaging"

# Test matrices
declare -A RPM_DISTROS=(
    ["centos:stream8"]="dnf"
    ["centos:stream9"]="dnf"
    ["fedora:38"]="dnf"
    ["fedora:39"]="dnf"
    ["rockylinux:8"]="dnf"
    ["rockylinux:9"]="dnf"
    ["almalinux:8"]="dnf"
    ["almalinux:9"]="dnf"
    ["opensuse/leap:15.5"]="zypper"
)

declare -A DEB_DISTROS=(
    ["ubuntu:20.04"]="apt"
    ["ubuntu:22.04"]="apt"
    ["ubuntu:23.04"]="apt"
    ["debian:11"]="apt"
    ["debian:12"]="apt"
)

# Functions
print_header() {
    echo -e "${BOLD}${BLUE}╔══════════════════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BOLD}${BLUE}║                    CloudWorkstation Package Testing                          ║${NC}"
    echo -e "${BOLD}${BLUE}╠══════════════════════════════════════════════════════════════════════════════╣${NC}"
    echo -e "${BOLD}${BLUE}║${NC} Testing Linux package installation across distributions                   ${BOLD}${BLUE}║${NC}"
    echo -e "${BOLD}${BLUE}║${NC} Version: $VERSION                                                          ${BOLD}${BLUE}║${NC}"
    echo -e "${BOLD}${BLUE}╚══════════════════════════════════════════════════════════════════════════════╝${NC}"
    echo ""
}

print_step() {
    echo -e "${BOLD}${YELLOW}▶${NC} $1"
}

print_success() {
    echo -e "${GREEN}✅${NC} $1"
}

print_error() {
    echo -e "${RED}❌${NC} $1" >&2
}

print_warning() {
    echo -e "${YELLOW}⚠️${NC} $1"
}

# Validation functions
check_dependencies() {
    print_step "Checking testing dependencies..."
    
    local missing_deps=()
    
    command -v docker >/dev/null 2>&1 || missing_deps+=("docker")
    
    if [[ ${#missing_deps[@]} -gt 0 ]]; then
        print_error "Missing required dependencies: ${missing_deps[*]}"
        echo "Please install Docker and ensure it's running"
        exit 1
    fi
    
    # Check if Docker daemon is running
    if ! docker info >/dev/null 2>&1; then
        print_error "Docker daemon is not running or not accessible"
        exit 1
    fi
    
    print_success "All testing dependencies are available"
}

validate_packages() {
    print_step "Validating package files..."
    
    local missing_packages=()
    
    # Check for RPM packages
    if [[ "$TEST_TYPE" == "rpm" || "$TEST_TYPE" == "all" ]]; then
        local rpm_dir="$PROJECT_ROOT/dist/rpm"
        if [[ ! -d "$rpm_dir" ]] || [[ -z "$(find "$rpm_dir" -name "*.rpm" 2>/dev/null)" ]]; then
            missing_packages+=("RPM packages in $rpm_dir")
        fi
    fi
    
    # Check for DEB packages
    if [[ "$TEST_TYPE" == "deb" || "$TEST_TYPE" == "all" ]]; then
        local deb_dir="$PROJECT_ROOT/dist/deb"
        if [[ ! -d "$deb_dir" ]] || [[ -z "$(find "$deb_dir" -name "*.deb" 2>/dev/null)" ]]; then
            missing_packages+=("DEB packages in $deb_dir")
        fi
    fi
    
    if [[ ${#missing_packages[@]} -gt 0 ]]; then
        print_error "Missing package files: ${missing_packages[*]}"
        echo "Build packages first with: make package-linux"
        exit 1
    fi
    
    print_success "Package files validated"
}

# Test functions
test_rpm_package() {
    local distro="$1"
    local package_manager="$2"
    local test_name="rpm_${distro//[:\/]/_}"
    
    print_step "Testing RPM on $distro..."
    
    # Find RPM package
    local rpm_file
    rpm_file=$(find "$PROJECT_ROOT/dist/rpm" -name "*.rpm" | head -1)
    
    if [[ ! -f "$rpm_file" ]]; then
        print_error "No RPM package found for testing"
        return 1
    fi
    
    # Create test script
    local test_script
    test_script=$(cat << 'EOF'
#!/bin/bash
set -e

# Update package manager
case "$1" in
    dnf)
        dnf update -y
        dnf install -y systemd curl
        ;;
    zypper)
        zypper refresh
        zypper install -y systemd curl
        ;;
esac

# Install package
echo "Installing CloudWorkstation package..."
case "$1" in
    dnf)
        dnf install -y /tmp/package.rpm
        ;;
    zypper)
        zypper install -y /tmp/package.rpm
        ;;
esac

# Test installation
echo "Testing installation..."
which cws || exit 1
which cwsd || exit 1

# Test binary execution
cws --version || exit 1
cwsd --version || exit 1

# Check systemd service
systemctl status prism || echo "Service not automatically started (expected)"
systemctl is-enabled prism || echo "Service not enabled (may be expected)"

# Test service can be enabled
systemctl enable prism || exit 1

# Check configuration files
test -f /etc/prism/daemon.conf || exit 1
test -f /etc/prism/aws/config.template || exit 1
test -f /etc/prism/aws/credentials.template || exit 1

# Check directories
test -d /var/lib/prism || exit 1
test -d /var/log/prism || exit 1

# Check user creation
getent passwd prism || exit 1
getent group prism || exit 1

echo "✅ RPM package test passed"
EOF
)
    
    # Run test in Docker container
    local container_name="cws_test_${test_name}_$$"
    local exit_code=0
    
    {
        echo "Starting container: $distro"
        docker run --name "$container_name" \
            --privileged \
            -v "$rpm_file:/tmp/package.rpm:ro" \
            -v /sys/fs/cgroup:/sys/fs/cgroup:rw \
            --tmpfs /run --tmpfs /run/lock \
            "$distro" \
            bash -c "$test_script $package_manager"
    } > "$TEST_RESULTS_DIR/${test_name}.log" 2>&1 || exit_code=$?
    
    # Cleanup container
    docker rm -f "$container_name" >/dev/null 2>&1 || true
    
    if [[ $exit_code -eq 0 ]]; then
        print_success "RPM test passed on $distro"
        return 0
    else
        print_error "RPM test failed on $distro (see $TEST_RESULTS_DIR/${test_name}.log)"
        return 1
    fi
}

test_deb_package() {
    local distro="$1"
    local package_manager="$2"
    local test_name="deb_${distro//[:\/]/_}"
    
    print_step "Testing DEB on $distro..."
    
    # Find DEB package
    local deb_file
    deb_file=$(find "$PROJECT_ROOT/dist/deb" -name "*.deb" | head -1)
    
    if [[ ! -f "$deb_file" ]]; then
        print_error "No DEB package found for testing"
        return 1
    fi
    
    # Create test script
    local test_script
    test_script=$(cat << 'EOF'
#!/bin/bash
set -e

# Update package manager
export DEBIAN_FRONTEND=noninteractive
apt-get update
apt-get install -y systemd curl

# Install package
echo "Installing CloudWorkstation package..."
dpkg -i /tmp/package.deb || true
apt-get install -f -y

# Test installation
echo "Testing installation..."
which cws || exit 1
which cwsd || exit 1

# Test binary execution
cws --version || exit 1
cwsd --version || exit 1

# Check systemd service
systemctl status prism || echo "Service not automatically started (expected)"
systemctl is-enabled prism || echo "Service not enabled (may be expected)"

# Test service can be enabled
systemctl enable prism || exit 1

# Check configuration files
test -f /etc/prism/daemon.conf || exit 1
test -f /etc/prism/aws/config.template || exit 1
test -f /etc/prism/aws/credentials.template || exit 1

# Check directories
test -d /var/lib/prism || exit 1
test -d /var/log/prism || exit 1

# Check user creation
getent passwd prism || exit 1
getent group prism || exit 1

echo "✅ DEB package test passed"
EOF
)
    
    # Run test in Docker container
    local container_name="cws_test_${test_name}_$$"
    local exit_code=0
    
    {
        echo "Starting container: $distro"
        docker run --name "$container_name" \
            --privileged \
            -v "$deb_file:/tmp/package.deb:ro" \
            -v /sys/fs/cgroup:/sys/fs/cgroup:rw \
            --tmpfs /run --tmpfs /run/lock \
            "$distro" \
            bash -c "$test_script $package_manager"
    } > "$TEST_RESULTS_DIR/${test_name}.log" 2>&1 || exit_code=$?
    
    # Cleanup container
    docker rm -f "$container_name" >/dev/null 2>&1 || true
    
    if [[ $exit_code -eq 0 ]]; then
        print_success "DEB test passed on $distro"
        return 0
    else
        print_error "DEB test failed on $distro (see $TEST_RESULTS_DIR/${test_name}.log)"
        return 1
    fi
}

run_rpm_tests() {
    print_step "Running RPM package tests..."
    
    local passed=0
    local failed=0
    
    for distro in "${!RPM_DISTROS[@]}"; do
        local package_manager="${RPM_DISTROS[$distro]}"
        
        if test_rpm_package "$distro" "$package_manager"; then
            ((passed++))
        else
            ((failed++))
        fi
    done
    
    echo ""
    print_step "RPM Test Summary:"
    echo "  Passed: $passed"
    echo "  Failed: $failed"
    echo "  Total:  $((passed + failed))"
    
    return $failed
}

run_deb_tests() {
    print_step "Running DEB package tests..."
    
    local passed=0
    local failed=0
    
    for distro in "${!DEB_DISTROS[@]}"; do
        local package_manager="${DEB_DISTROS[$distro]}"
        
        if test_deb_package "$distro" "$package_manager"; then
            ((passed++))
        else
            ((failed++))
        fi
    done
    
    echo ""
    print_step "DEB Test Summary:"
    echo "  Passed: $passed"
    echo "  Failed: $failed"
    echo "  Total:  $((passed + failed))"
    
    return $failed
}

print_test_summary() {
    local total_failed="$1"
    
    echo ""
    if [[ $total_failed -eq 0 ]]; then
        echo -e "${BOLD}${GREEN}╔══════════════════════════════════════════════════════════════════════════════╗${NC}"
        echo -e "${BOLD}${GREEN}║                         All Package Tests Passed                           ║${NC}"
        echo -e "${BOLD}${GREEN}╚══════════════════════════════════════════════════════════════════════════════╝${NC}"
    else
        echo -e "${BOLD}${RED}╔══════════════════════════════════════════════════════════════════════════════╗${NC}"
        echo -e "${BOLD}${RED}║                        Some Package Tests Failed                            ║${NC}"
        echo -e "${BOLD}${RED}╚══════════════════════════════════════════════════════════════════════════════╝${NC}"
    fi
    
    echo ""
    echo -e "${BOLD}📁 Test Results:${NC}"
    echo "   Directory: $TEST_RESULTS_DIR"
    echo "   Log files: $(find "$TEST_RESULTS_DIR" -name "*.log" 2>/dev/null | wc -l)"
    echo ""
    
    if [[ $total_failed -gt 0 ]]; then
        echo -e "${BOLD}🔍 Failed Tests:${NC}"
        find "$TEST_RESULTS_DIR" -name "*.log" -exec grep -L "✅.*test passed" {} \; | while read -r log_file; do
            echo "   $(basename "$log_file" .log)"
        done
        echo ""
    fi
}

cleanup_containers() {
    print_step "Cleaning up test containers..."
    docker ps -a --filter "name=cws_test_" --format "{{.Names}}" | xargs -r docker rm -f >/dev/null 2>&1 || true
}

# Main execution
main() {
    local TEST_TYPE="all"
    local total_failed=0
    
    # Handle command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --rpm)
                TEST_TYPE="rpm"
                shift
                ;;
            --deb)
                TEST_TYPE="deb"
                shift
                ;;
            --help|-h)
                echo "CloudWorkstation Linux Package Testing Script"
                echo ""
                echo "Usage: $0 [OPTIONS]"
                echo ""
                echo "Options:"
                echo "  --rpm        Test RPM packages only"
                echo "  --deb        Test DEB packages only"
                echo "  --help       Show this help message"
                echo ""
                echo "Environment Variables:"
                echo "  VERSION      Package version to test (default: $VERSION)"
                echo ""
                echo "Tested Distributions:"
                echo ""
                echo "RPM Distributions:"
                for distro in "${!RPM_DISTROS[@]}"; do
                    echo "  $distro (${RPM_DISTROS[$distro]})"
                done
                echo ""
                echo "DEB Distributions:"
                for distro in "${!DEB_DISTROS[@]}"; do
                    echo "  $distro (${DEB_DISTROS[$distro]})"
                done
                echo ""
                exit 0
                ;;
            *)
                print_error "Unknown option: $1"
                echo "Use --help for usage information"
                exit 1
                ;;
        esac
    done
    
    # Set up exit trap
    trap cleanup_containers EXIT
    
    # Create test results directory
    mkdir -p "$TEST_RESULTS_DIR"
    
    # Execute test pipeline
    print_header
    check_dependencies
    validate_packages
    
    # Run tests based on type
    case "$TEST_TYPE" in
        rpm)
            run_rpm_tests || total_failed=$?
            ;;
        deb)
            run_deb_tests || total_failed=$?
            ;;
        all)
            run_rpm_tests || total_failed=$((total_failed + $?))
            run_deb_tests || total_failed=$((total_failed + $?))
            ;;
    esac
    
    print_test_summary $total_failed
    
    if [[ $total_failed -eq 0 ]]; then
        print_success "All package tests completed successfully!"
        exit 0
    else
        print_error "$total_failed test(s) failed. Check log files for details."
        exit 1
    fi
}

# Execute main function
main "$@"