#!/usr/bin/env bash
# Download cl100k_base.tiktoken for end-to-end testing.
# Synthetic test data lives in testdata/ and is committed directly.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
TESTDATA="$SCRIPT_DIR/../testdata"
mkdir -p "$TESTDATA"

CL100K="$TESTDATA/cl100k_base.tiktoken"
if [ -f "$CL100K" ]; then
    echo "[skip] $CL100K already exists"
else
    URL="https://openaipublic.blob.core.windows.net/encodings/cl100k_base.tiktoken"
    echo "Downloading $URL ..."
    curl -sSL -o "$CL100K" "$URL"
    echo "Saved to $CL100K"
fi
