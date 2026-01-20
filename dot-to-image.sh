#!/bin/bash
# Convert DOT file to various image formats using Graphviz

DOT_FILE=${1:-kafka-relationships.dot}
OUTPUT_PREFIX=${2:-kafka-relationships}

if [ ! -f "$DOT_FILE" ]; then
    echo "Error: DOT file '$DOT_FILE' not found"
    echo "Usage: $0 <dot-file> [output-prefix]"
    exit 1
fi

# Check if Graphviz is installed
if ! command -v dot &> /dev/null; then
    echo "Graphviz is not installed!"
    echo ""
    echo "To install:"
    echo "  macOS:   brew install graphviz"
    echo "  Ubuntu:  sudo apt-get install graphviz"
    echo "  Windows: choco install graphviz"
    echo ""
    echo "Or download from: https://graphviz.org/download/"
    exit 1
fi

echo "Converting $DOT_FILE..."

# Generate PNG (high quality)
echo "  → ${OUTPUT_PREFIX}.png"
dot -Tpng -Gdpi=300 "$DOT_FILE" -o "${OUTPUT_PREFIX}.png"

# Generate SVG (scalable, best for web)
echo "  → ${OUTPUT_PREFIX}.svg"
dot -Tsvg "$DOT_FILE" -o "${OUTPUT_PREFIX}.svg"

# Generate PDF (best for printing)
echo "  → ${OUTPUT_PREFIX}.pdf"
dot -Tpdf "$DOT_FILE" -o "${OUTPUT_PREFIX}.pdf"

echo ""
echo "✓ Done! Generated:"
ls -lh "${OUTPUT_PREFIX}".{png,svg,pdf} 2>/dev/null || true

echo ""
echo "View with:"
echo "  open ${OUTPUT_PREFIX}.png      # macOS"
echo "  xdg-open ${OUTPUT_PREFIX}.png  # Linux"
echo "  start ${OUTPUT_PREFIX}.png     # Windows"
