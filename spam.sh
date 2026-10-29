#!/usr/bin/env bash
set -euo pipefail

TARGET_URL="http://localhost:8080/hello"
TOTAL_REQUESTS=1

generate_requests() {
  for ((i = 1; i <= TOTAL_REQUESTS; i++)); do
    echo "url = \"${TARGET_URL}\""
    echo "silent"
    echo "output = \"/dev/null\""
  done
}

generate_requests | curl -K -