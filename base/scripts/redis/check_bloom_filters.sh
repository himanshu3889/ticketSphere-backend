#!/bin/bash
# scripts/check_bloom_filters.sh

set -e

check_filter() {
    local key=$1
    if redis-cli BF.INFO "$key" > /dev/null 2>&1; then
        echo "OK: $key"
    else
        echo "FAIL: $key missing"
        exit 1
    fi
}

check_filter "ticketspehere:bloom:user:id"
check_filter "ticketspehere:bloom:email"