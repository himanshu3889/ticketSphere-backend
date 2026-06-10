#!/bin/bash

set -e

REDIS_PASSWORD="${REDIS_PASSWORD}"

run() {
    docker exec -e REDISCLI_AUTH="$REDIS_PASSWORD" ticketsphere-redis redis-cli "$@"
}

check_filter() {
    local key=$1
    if run BF.INFO "$key" > /dev/null 2>&1; then
        echo "OK: $key"
    else
        echo "FAIL: $key missing"
    fi
}

check_filter "ticketsphere:bloom:user:id"
check_filter "ticketsphere:bloom:user:email"
