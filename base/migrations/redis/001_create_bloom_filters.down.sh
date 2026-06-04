#!/bin/bash

set -e

REDIS_PASSWORD="${REDIS_PASSWORD}"

run() {
    docker exec -e REDISCLI_AUTH="$REDIS_PASSWORD" ticketsphere-redis redis-cli "$@"
}

# A helper function to cleanly handle deletion
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

echo "Reverting Bloom filters..."

delete_filter "ticketspehere:bloom:user:id"
delete_filter "ticketspehere:bloom:user:email"

echo "Bloom filter migration 001 reverted!"