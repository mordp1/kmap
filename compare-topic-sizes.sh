#!/bin/bash
# Compare topic sizes between two snapshots or clusters

set -e

if [ $# -ne 2 ]; then
    cat << EOF
Usage: $0 <file1.json> <file2.json>

Compare topic sizes between two kmap topic-sizes JSON reports.

EXAMPLES:
    # Compare before and after
    $0 sizes-before.json sizes-after.json
    
    # Compare two clusters
    $0 cluster1-sizes.json cluster2-sizes.json

OUTPUT:
    - Topics that increased in size
    - Topics that decreased in size
    - New topics
    - Removed topics
    - Total size change

EOF
    exit 1
fi

FILE1="$1"
FILE2="$2"

if [ ! -f "$FILE1" ]; then
    echo "Error: File not found: $FILE1"
    exit 1
fi

if [ ! -f "$FILE2" ]; then
    echo "Error: File not found: $FILE2"
    exit 1
fi

# Check if jq is installed
if ! command -v jq &> /dev/null; then
    echo "Error: jq is required but not installed"
    echo "Install with: brew install jq  (macOS) or apt-get install jq  (Linux)"
    exit 1
fi

echo "================================================================================"
echo "Topic Size Comparison"
echo "================================================================================"
echo ""
echo "File 1: $FILE1"
jq -r '"  Cluster: \(.cluster)\n  Timestamp: \(.timestamp)\n  Total: \(.total_size_human) (\(.total_topics) topics)"' "$FILE1"
echo ""
echo "File 2: $FILE2"
jq -r '"  Cluster: \(.cluster)\n  Timestamp: \(.timestamp)\n  Total: \(.total_size_human) (\(.total_topics) topics)"' "$FILE2"
echo ""
echo "================================================================================"

# Get totals
TOTAL1=$(jq -r '.total_size_bytes' "$FILE1")
TOTAL2=$(jq -r '.total_size_bytes' "$FILE2")
DIFF=$((TOTAL2 - TOTAL1))

# Format bytes to human readable
format_bytes() {
    local bytes=$1
    local sign=""
    if [ $bytes -lt 0 ]; then
        sign="-"
        bytes=$((bytes * -1))
    fi
    
    if [ $bytes -lt 1024 ]; then
        echo "${sign}${bytes} B"
    elif [ $bytes -lt 1048576 ]; then
        echo "${sign}$(echo "scale=2; $bytes / 1024" | bc) KiB"
    elif [ $bytes -lt 1073741824 ]; then
        echo "${sign}$(echo "scale=2; $bytes / 1048576" | bc) MiB"
    elif [ $bytes -lt 1099511627776 ]; then
        echo "${sign}$(echo "scale=2; $bytes / 1073741824" | bc) GiB"
    else
        echo "${sign}$(echo "scale=2; $bytes / 1099511627776" | bc) TiB"
    fi
}

echo ""
echo "TOTAL SIZE CHANGE:"
echo "  Before: $(format_bytes $TOTAL1)"
echo "  After:  $(format_bytes $TOTAL2)"
echo "  Diff:   $(format_bytes $DIFF)"
if [ $DIFF -gt 0 ]; then
    PERCENT=$(echo "scale=2; ($DIFF * 100) / $TOTAL1" | bc)
    echo "  Change: +${PERCENT}% (increase)"
elif [ $DIFF -lt 0 ]; then
    PERCENT=$(echo "scale=2; (${DIFF#-} * 100) / $TOTAL1" | bc)
    echo "  Change: -${PERCENT}% (decrease)"
else
    echo "  Change: No change"
fi

echo ""
echo "================================================================================"
echo "TOP 10 SIZE INCREASES:"
echo "================================================================================"

# Create temporary files for comparison
TMP1=$(mktemp)
TMP2=$(mktemp)
trap "rm -f $TMP1 $TMP2" EXIT

jq -r '.topics[] | "\(.topic)|\(.total_size_bytes)"' "$FILE1" | sort > "$TMP1"
jq -r '.topics[] | "\(.topic)|\(.total_size_bytes)"' "$FILE2" | sort > "$TMP2"

# Find topics in both files and calculate differences
join -t '|' "$TMP1" "$TMP2" | while IFS='|' read -r topic size1 size2; do
    diff=$((size2 - size1))
    if [ $diff -gt 0 ]; then
        echo "$diff|$topic|$(format_bytes $diff)"
    fi
done | sort -t '|' -k1 -rn | head -10 | while IFS='|' read -r diff topic formatted; do
    echo "  $topic: +$formatted"
done

echo ""
echo "================================================================================"
echo "TOP 10 SIZE DECREASES:"
echo "================================================================================"

join -t '|' "$TMP1" "$TMP2" | while IFS='|' read -r topic size1 size2; do
    diff=$((size2 - size1))
    if [ $diff -lt 0 ]; then
        echo "${diff#-}|$topic|$(format_bytes ${diff#-})"
    fi
done | sort -t '|' -k1 -rn | head -10 | while IFS='|' read -r diff topic formatted; do
    echo "  $topic: -$formatted"
done

echo ""
echo "================================================================================"
echo "NEW TOPICS (in File 2, not in File 1):"
echo "================================================================================"

comm -13 <(jq -r '.topics[].topic' "$FILE1" | sort) <(jq -r '.topics[].topic' "$FILE2" | sort) | while read -r topic; do
    size=$(jq -r --arg topic "$topic" '.topics[] | select(.topic == $topic) | .total_size_human' "$FILE2")
    echo "  $topic: $size"
done

echo ""
echo "================================================================================"
echo "REMOVED TOPICS (in File 1, not in File 2):"
echo "================================================================================"

comm -23 <(jq -r '.topics[].topic' "$FILE1" | sort) <(jq -r '.topics[].topic' "$FILE2" | sort) | while read -r topic; do
    size=$(jq -r --arg topic "$topic" '.topics[] | select(.topic == $topic) | .total_size_human' "$FILE1")
    echo "  $topic: $size"
done

echo ""
echo "================================================================================"
