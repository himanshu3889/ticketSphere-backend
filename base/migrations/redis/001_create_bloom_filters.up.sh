#!/bin/bash

set -e

REDIS_PASSWORD="${REDIS_PASSWORD}"

run() {
    docker exec -e REDISCLI_AUTH="$REDIS_PASSWORD" ticketsphere-redis redis-cli "$@"
}

# A helper function to cleanly handle the Bloom filter creation
create_filter() {
    local filter_name="$1"
    local error_rate="$2"
    local capacity="$3"

    # Capture the raw output from Redis (and strip any hidden Windows carriage returns)
    local result
    result=$(run BF.RESERVE "$filter_name" "$error_rate" "$capacity" | tr -d '\r')

    # Check what Redis actually said and print a nice message
    if [[ "$result" == "OK" ]]; then
        echo " [+] Created Bloom filter: $filter_name"
    elif [[ "$result" == *"exists"* ]]; then
        echo " [~] Filter already exists:  $filter_name (Skipped)"
    else
        echo " [!] Failed to create $filter_name: $result"
        exit 1
    fi
}

echo "Configuring Bloom filters..."

create_filter "ticketspehere:bloom:user:id" 0.001 1000000
create_filter "ticketspehere:bloom:user:email" 0.001 500000

echo "Bloom filter migration 001 complete!"