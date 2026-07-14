#!/bin/bash
set -e

# Ensure REDIS_PASSWORD is available (assuming you load .env before running this)
REDIS_PASSWORD="${REDIS_PASSWORD}"

# Helper to run commands in the Redis container
run() {
    docker exec -e REDISCLI_AUTH="$REDIS_PASSWORD" ticketsphere-redis redis-cli "$@"
}

# Helper for creating bloom filter
create_filter() {
    local filter_name="$1"
    local error_rate="$2"
    local capacity="$3"

    local result
    result=$(run BF.RESERVE "$filter_name" "$error_rate" "$capacity" | tr -d '\r')

    if [[ "$result" == "OK" ]]; then
        echo " [+] Created Bloom filter: $filter_name"
    elif [[ "$result" == *"exists"* ]]; then
        echo " [~] Filter already exists:  $filter_name (Skipped)"
    else
        echo " [!] Failed to create $filter_name: $result"
        exit 1
    fi
}

# Helper function for deleting bloom filter
delete_filter() {
    local filter_name="$1"
    
    # Capture the raw output from Redis (and strip Windows carriage returns)
    local result
    result=$(run DEL "$filter_name" | tr -d '\r')

    # Check if Redis returned 1 (deleted) or 0 (not found)
    if [[ "$result" == *"1"* ]]; then
        echo " [-] Removed Bloom filter: $filter_name"
    elif [[ "$result" == *"0"* ]]; then
        echo " [~] Filter not found:     $filter_name (Skipped)"
    else
        echo " [!] Failed to remove $filter_name: $result"
    fi
}
