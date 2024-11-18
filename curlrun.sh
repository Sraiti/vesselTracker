#!/bin/bash

# load .env var to child process
#set -a && source .env && set +a

# Base URL for the API
BASE_URL="localhost:8080"

# Common headers
CONTENT_TYPE="Content-Type: application/json"

# Helper function to build URL with parameters
build_url() {
    local endpoint=$1
    local params=$2
    
    if [ -z "$params" ]; then
        echo "${BASE_URL}${endpoint}"
    else
        # Remove any leading '?' or '&' from params
        params=$(echo "$params" | sed 's/^[?&]*//')
        echo "${BASE_URL}${endpoint}?${params}"
    fi
}

# Helper function for making requests
make_request() {
    local method=$1
    local endpoint=$2
    local params=$3
    local data=$4

    # Build full URL with parameters
    local full_url=$(build_url "$endpoint" "$params")

    if [ -n "$data" ]; then
        curl -X $method \
             -H "$CONTENT_TYPE" \
             -H "$AUTH_HEADER" \
             -d "$data" \
             "$full_url"
    else
        curl -X $method \
             -H "$CONTENT_TYPE" \
             -H "$AUTH_HEADER" \
             "$full_url"
    fi
    echo ""
}


search_vessels() {
    local params="OriginPortUnLoCode=$1&DestinationPortUnLoCode=$2&Destination=$3&Origin=Value&DepartureDate=$4"
    echo "Searching vessels..."
    make_request "POST" "/search" "$params"
}


# Example params:
#
# search vessels => "OriginPortUnLoCode=Value&DestinationPortUnLoCode=Value&Destination=Value&Origin=Value&DepartureDate=Value"
#
