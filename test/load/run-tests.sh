#!/bin/bash
# Load Testing Runner for DID Gateway
# Phase 3: Scalability & Performance

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
GATEWAY_HOST="${GATEWAY_HOST:-localhost:8080}"
K6_VERSION="0.48.0"
RESULTS_DIR="./results"

echo -e "${BLUE}=================================${NC}"
echo -e "${BLUE}DID Gateway Load Testing${NC}"
echo -e "${BLUE}=================================${NC}"
echo ""

# Check if k6 is installed
if ! command -v k6 &> /dev/null; then
    echo -e "${YELLOW}k6 not found. Installing...${NC}"
    
    # Detect OS
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        sudo gpg -k
        sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
        echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
        sudo apt-get update
        sudo apt-get install k6
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        brew install k6
    else
        echo -e "${RED}Unsupported OS. Please install k6 manually: https://k6.io/docs/getting-started/installation${NC}"
        exit 1
    fi
fi

# Create results directory
mkdir -p "$RESULTS_DIR"

# Function to run a test
run_test() {
    local test_name=$1
    local test_file=$2
    local duration=$3
    
    echo ""
    echo -e "${GREEN}Running: $test_name${NC}"
    echo -e "${BLUE}Duration: $duration${NC}"
    echo -e "${BLUE}Target: $GATEWAY_HOST${NC}"
    echo ""
    
    # Run k6 test
    GATEWAY_HOST="$GATEWAY_HOST" k6 run \
        --out json="$RESULTS_DIR/${test_name}.json" \
        "$test_file"
    
    local exit_code=$?
    
    if [ $exit_code -eq 0 ]; then
        echo -e "${GREEN}✓ $test_name completed successfully${NC}"
    else
        echo -e "${RED}✗ $test_name failed (exit code: $exit_code)${NC}"
    fi
    
    return $exit_code
}

# Function to generate HTML report
generate_report() {
    echo ""
    echo -e "${BLUE}Generating HTML reports...${NC}"
    
    for json_file in "$RESULTS_DIR"/*.json; do
        if [ -f "$json_file" ]; then
            base_name=$(basename "$json_file" .json)
            echo "Converting $base_name..."
            
            # Use k6-reporter or custom script
            # For now, just create summary
            cat "$json_file" | jq -r '
                select(.type=="Point" and .metric=="http_req_duration") |
                .data.value
            ' | awk '{
                sum+=$1; count++; 
                if(min==""){min=max=$1}; 
                if($1>max){max=$1}; 
                if($1<min){min=$1}
            } END {
                print "Summary for '"$base_name"':"
                print "  Count:", count
                print "  Mean:", sum/count "ms"
                print "  Min:", min "ms"
                print "  Max:", max "ms"
            }' > "$RESULTS_DIR/${base_name}_summary.txt"
        fi
    done
    
    echo -e "${GREEN}Reports generated in $RESULTS_DIR/${NC}"
}

# Main menu
echo "Select test to run:"
echo "  1) Auth Flow (Full load test)"
echo "  2) DID Resolution (Cache performance)"
echo "  3) Quick smoke test (1 min)"
echo "  4) All tests"
echo "  5) Generate reports from existing results"
echo ""
read -p "Enter choice [1-5]: " choice

case $choice in
    1)
        run_test "auth-flow" "auth-flow.js" "17m"
        generate_report
        ;;
    2)
        run_test "did-resolution" "did-resolution.js" "10m"
        generate_report
        ;;
    3)
        echo -e "${YELLOW}Running quick smoke test...${NC}"
        GATEWAY_HOST="$GATEWAY_HOST" k6 run \
            --vus 100 \
            --duration 1m \
            auth-flow.js
        ;;
    4)
        echo -e "${GREEN}Running all tests...${NC}"
        run_test "auth-flow" "auth-flow.js" "17m"
        sleep 60  # Cool down
        run_test "did-resolution" "did-resolution.js" "10m"
        generate_report
        ;;
    5)
        generate_report
        ;;
    *)
        echo -e "${RED}Invalid choice${NC}"
        exit 1
        ;;
esac

echo ""
echo -e "${GREEN}=================================${NC}"
echo -e "${GREEN}Load testing complete!${NC}"
echo -e "${GREEN}=================================${NC}"
echo ""
echo "Results saved to: $RESULTS_DIR/"
echo ""
echo "Next steps:"
echo "  1. Review results in $RESULTS_DIR/"
echo "  2. Check Grafana dashboards for detailed metrics"
echo "  3. Analyze p99 latency and error rates"
echo "  4. Tune cache sizes or HPA thresholds if needed"
echo ""
