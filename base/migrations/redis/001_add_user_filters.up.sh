#!/bin/bash

# Load the shared functions
# Figure out exactly where this specific script lives
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Source the helpers using that absolute directory path
if [ -f "$SCRIPT_DIR/bloom_helpers.sh" ]; then
    source "$SCRIPT_DIR/bloom_helpers.sh"
else
    echo "Error: bloom_helpers.sh not found at $SCRIPT_DIR/bloom_helpers.sh"
    exit 1
fi

echo "001 migration: Configuring initial user Bloom filters..."

create_filter "ticketsphere:bloom:user:id" 0.01 1000000
create_filter "ticketsphere:bloom:user:email" 0.01 1000000

echo "User bloom filter 001 migration complete!"