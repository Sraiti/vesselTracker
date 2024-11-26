#!/bin/bash

# Save as benchmark.sh
# Configuration
NUM_REQUESTS=10
CONCURRENT=2

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Your specific request - note the careful handling of quotes
CURL_CMD="curl --location 'http://localhost:3058/search' \
  --header 'Content-Type: application/json' \
  --header 'Cookie: x-auth-token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1aWQiOiI2NmI0YzY5MmU0YjdjNzBjYjRkNTQ2YzQiLCJjaWQiOiI2NmZiZmZmMWUyNGEyZjI2M2QyN2Q3NGMiLCJzZXNzaW9uIjp7ImlwIjoiMTI3LjAuMC4xIiwidXNlckFnZW50IjoiUG9zdG1hblJ1bnRpbWUvNy40Mi4wIiwiZGF0ZSI6IjIwMjQtMTEtMjVUMTQ6MDQ6MzguODY4WiIsImtleSI6InFzaHRpdGNnMW9qIn0sImlhdCI6MTczMjU0MzQ3OCwiZXhwIjoxNzMyNjI5ODc4fQ.xjO-YqoGKi3mVHGjfHW9Z4TUFNGYOyviDlkDeH9LY-Q' \
  --data '{
    \"OriginPortUnLoCode\": \"MAPTM\",
    \"DestinationPortUnLoCode\": \"CNSGH\",
    \"Origin\": \"Port Tangier Mediterranee\",
    \"Destination\": \"Shanghai\",
    \"DepartureDate\": \"2024-04-01T00:00:00\"
  }'"

echo -e "${BLUE}Running Search API benchmark...${NC}"
echo -e "Endpoint: http://localhost:3058/search"
echo -e "Route: Port Tangier Mediterranee → Shanghai"
echo -e "Requests: $NUM_REQUESTS, Concurrent: $CONCURRENT\n"

# Function to format milliseconds
format_time() {
    if (( $(echo "$1 >= 1000" | bc -l) )); then
        printf "%.2f s" $(echo "$1/1000" | bc -l)
    else
        printf "%.2f ms" "$1"
    fi
}

# Run benchmark
for i in $(seq 1 $NUM_REQUESTS); do
    echo -n "Request $i: "
    
    # Measure time and execute curl
    start=$(date +%s.%N)
    response=$(eval $CURL_CMD)
    end=$(date +%s.%N)
    
    # Calculate duration in milliseconds
    duration=$(echo "($end - $start) * 1000" | bc)
    
    # Get response size
    size=${#response}
    
    # Check if response contains error
    if [[ $response == *"error"* ]]; then
        echo -e "${RED}$(format_time $duration)${NC} - Size: $size bytes - Error"
        echo "Response: $response"
    else
        echo -e "${GREEN}$(format_time $duration)${NC} - Size: $size bytes"
    fi
    
    # Store timing for statistics
    echo "$duration" >> times.txt
    
    # Add a small delay between requests
    sleep 0.1
done

# Calculate statistics
echo -e "\n${BLUE}Performance Statistics:${NC}"
TIMES=$(cat times.txt)
MIN=$(echo "$TIMES" | sort -n | head -1)
MAX=$(echo "$TIMES" | sort -n | tail -1)
AVG=$(echo "$TIMES" | awk '{ sum += $1 } END { print sum/NR }')
MEDIAN=$(echo "$TIMES" | sort -n | awk '{a[NR]=$1} END {print (NR%2==1)?a[int(NR/2)+1]:(a[NR/2]+a[NR/2+1])/2}')
P95=$(echo "$TIMES" | sort -n | awk '{a[NR]=$1} END {print a[int(NR*0.95)]}')
STDDEV=$(echo "$TIMES" | awk '{sum+=$1; sumsq+=$1*$1}END{print sqrt(sumsq/NR - (sum/NR)**2)}')

echo "┌────────────────────────────────────────┐"
echo "│ Minimum response time: $(format_time $MIN)"
echo "│ Maximum response time: $(format_time $MAX)"
echo "│ Average response time: $(format_time $AVG)"
echo "│ Median response time:  $(format_time $MEDIAN)"
echo "│ 95th percentile:      $(format_time $P95)"
echo "│ Standard deviation:   $(format_time $STDDEV)"
echo "└────────────────────────────────────────┘"

# Count successful and error responses
TOTAL=$(wc -l < times.txt)
ERROR_COUNT=$(grep -c "error" times.txt || true)
SUCCESS_COUNT=$((TOTAL - ERROR_COUNT))

echo -e "\n${BLUE}Response Statistics:${NC}"
echo "┌────────────────────────────────────────┐"
echo "│ Total requests:    $TOTAL"
echo "│ Successful:        $SUCCESS_COUNT"
echo "│ Errors:           $ERROR_COUNT"
echo "└────────────────────────────────────────┘"

# Cleanup
rm times.txt
