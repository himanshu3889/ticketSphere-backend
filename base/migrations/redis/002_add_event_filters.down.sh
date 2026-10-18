#!/bin/bash

# Figure out exactly where this specific script lives
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Source the helpers using that absolute directory path
if [ -f "$SCRIPT_DIR/bloom_helpers.sh" ]; then
    source "$SCRIPT_DIR/bloom_helpers.sh"
else
    echo "Error: bloom_helpers.sh not found at $SCRIPT_DIR/bloom_helpers.sh"
    exit 1
fi

echo "Reverting 002 migration: Removing event bloom filters..."

delete_filter "ticketsphere:bloom:event:id"

echo "Event bloom filter 002 reverted!"