#!/bin/bash
# scripts/prod.sh — Download all OpenAI tiktoken files and run 24h GA production builds.
set -euo pipefail

CDN="https://openaipublic.blob.core.windows.net/encodings"
DATA_DIR="proddata"
OUT_DIR="prodout"

# All significant OpenAI tiktoken encodings.
# p50k_edit uses the same .tiktoken as p50k_base but different metadata.
ENCODINGS=(
    "cl100k_base"
    "o200k_base"
    "p50k_base"
    "r50k_base"
)

GA_TIME="${GA_TIME:-24h}"

mkdir -p "$DATA_DIR" "$OUT_DIR"

# --- Step 1: Generate JSON metadata for each encoding ---
echo "=== Generating JSON metadata ==="
go run scripts/gen_testdata.go

# Copy the generated metadata JSONs from testdata/ to proddata/.
for enc in "${ENCODINGS[@]}"; do
    cp "testdata/${enc}.json" "${DATA_DIR}/${enc}.json"
    echo "  Copied ${enc}.json"
done
# p50k_edit shares p50k_base.tiktoken but has its own metadata.
cp "testdata/p50k_edit.json" "${DATA_DIR}/p50k_edit.json"
echo "  Copied p50k_edit.json"

# --- Step 2: Download .tiktoken files ---
echo ""
echo "=== Downloading .tiktoken files ==="
for enc in "${ENCODINGS[@]}"; do
    dest="${DATA_DIR}/${enc}.tiktoken"
    if [ -f "$dest" ]; then
        echo "  ${enc}.tiktoken already exists, skipping."
    else
        echo "  Downloading ${enc}.tiktoken ..."
        curl -fSL -o "$dest" "${CDN}/${enc}.tiktoken"
        echo "  Done: $(wc -l < "$dest") lines."
    fi
done

# --- Step 3: Run scs build on each encoding ---
echo ""
echo "=== Running production builds (GA budget: ${GA_TIME}) ==="

# Build the scs binary once.
echo "  Building scs binary..."
go build -o scs .

run_build() {
    local enc="$1"
    local tiktoken="${DATA_DIR}/${enc}.tiktoken"
    local metadata="${DATA_DIR}/${enc}.json"
    local output="${OUT_DIR}/${enc}.scs"

    echo ""
    echo "--- ${enc} ---"
    echo "  Input:    ${tiktoken} ($(wc -l < "$tiktoken") tokens)"
    echo "  Metadata: ${metadata}"
    echo "  Output:   ${output}"
    echo "  GA time:  ${GA_TIME}"
    echo ""

    ./scs build \
        --tiktoken \
        -i "$tiktoken" \
        -o "$output" \
        --metadata "$metadata" \
        --ga-time "$GA_TIME" \
        -v

    echo ""
    echo "  ✓ ${enc} complete."
    echo "  Output files:"
    ls -lh "${OUT_DIR}/${enc}".*
    echo ""
}

for enc in "${ENCODINGS[@]}"; do
    run_build "$enc"
done

# p50k_edit: same .tiktoken as p50k_base, different metadata.
echo ""
echo "--- p50k_edit (shares p50k_base.tiktoken) ---"
./scs build \
    --tiktoken \
    -i "${DATA_DIR}/p50k_base.tiktoken" \
    -o "${OUT_DIR}/p50k_edit.scs" \
    --metadata "${DATA_DIR}/p50k_edit.json" \
    --ga-time "$GA_TIME" \
    -v
echo "  ✓ p50k_edit complete."
ls -lh "${OUT_DIR}/p50k_edit".*

echo ""
echo "=== All production builds complete ==="
echo "Output directory: ${OUT_DIR}/"
ls -lhS "${OUT_DIR}/"
