#!/bin/bash
set -e

# Ensure we are in the project root
cd "$(dirname "$0")/.."

echo "Generating code coverage report..."

# Run llvm-cov with summary-only to get file-level granularity
# --text: outputs a plain text report
# --output-path: where to save the detailed report (optional, we use stdout here)
# We redirect the summary to COVERAGE.txt
# We use --color never to avoid escape codes in the committed file
cargo llvm-cov --summary-only --color never > COVERAGE.txt

echo "Coverage report updated in COVERAGE.txt"
