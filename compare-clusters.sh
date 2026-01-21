#!/bin/bash
# Kafka Cluster Comparison Script
# Compare two clusters to validate migration

if [ "$#" -ne 2 ]; then
    echo "Usage: $0 <cluster1.json> <cluster2.json>"
    echo ""
    echo "Example:"
    echo "  $0 source-cluster.json target-cluster.json"
    exit 1
fi

SOURCE=$1
TARGET=$2

if [ ! -f "$SOURCE" ]; then
    echo "Error: Source file $SOURCE not found"
    exit 1
fi

if [ ! -f "$TARGET" ]; then
    echo "Error: Target file $TARGET not found"
    exit 1
fi

echo "=========================================="
echo "Kafka Cluster Migration Comparison"
echo "=========================================="
echo ""

# Extract metrics
SOURCE_TOPICS=$(jq '.total_topics' "$SOURCE")
TARGET_TOPICS=$(jq '.total_topics' "$TARGET")

SOURCE_PARTITIONS=$(jq '.total_partitions' "$SOURCE")
TARGET_PARTITIONS=$(jq '.total_partitions' "$TARGET")

SOURCE_MESSAGES=$(jq '.total_messages' "$SOURCE")
TARGET_MESSAGES=$(jq '.total_messages' "$TARGET")

SOURCE_BROKERS=$(jq '.brokers | length' "$SOURCE")
TARGET_BROKERS=$(jq '.brokers | length' "$TARGET")

echo "📊 Cluster Overview:"
echo "-------------------"
printf "%-20s %15s %15s %15s\n" "Metric" "Source" "Target" "Difference"
echo "-------------------------------------------------------------------"

# Topics
TOPIC_DIFF=$((TARGET_TOPICS - SOURCE_TOPICS))
printf "%-20s %15s %15s %15s\n" "Topics" "$SOURCE_TOPICS" "$TARGET_TOPICS" "$TOPIC_DIFF"

# Partitions
PARTITION_DIFF=$((TARGET_PARTITIONS - SOURCE_PARTITIONS))
printf "%-20s %15s %15s %15s\n" "Partitions" "$SOURCE_PARTITIONS" "$TARGET_PARTITIONS" "$PARTITION_DIFF"

# Messages
MESSAGE_DIFF=$((TARGET_MESSAGES - SOURCE_MESSAGES))
MESSAGE_PERCENT=$(echo "scale=2; ($TARGET_MESSAGES / $SOURCE_MESSAGES) * 100" | bc)
printf "%-20s %15s %15s %15s\n" "Messages" "$SOURCE_MESSAGES" "$TARGET_MESSAGES" "$MESSAGE_DIFF"

# Brokers
BROKER_DIFF=$((TARGET_BROKERS - SOURCE_BROKERS))
printf "%-20s %15s %15s %15s\n" "Brokers" "$SOURCE_BROKERS" "$TARGET_BROKERS" "$BROKER_DIFF"

echo ""
echo "📈 Data Completeness:"
echo "--------------------"
printf "Message Replication: %.2f%%\n" "$MESSAGE_PERCENT"

if [ "$MESSAGE_PERCENT" = "100.00" ]; then
    echo "✅ Perfect match! All messages replicated"
elif (( $(echo "$MESSAGE_PERCENT >= 99.0" | bc -l) )); then
    echo "✅ Excellent! Migration is >99% complete"
elif (( $(echo "$MESSAGE_PERCENT >= 95.0" | bc -l) )); then
    echo "⚠️  Good, but check for missing data (>95%)"
else
    echo "❌ Warning: Significant data loss detected (<95%)"
fi

echo ""
echo "🔍 Topic-by-Topic Comparison:"
echo "-----------------------------"

# Compare topic message counts
jq -r '.topics[] | "\(.name)|\(.partitions)|\(.total_messages)"' "$SOURCE" > /tmp/source_topics.txt
jq -r '.topics[] | "\(.name)|\(.partitions)|\(.total_messages)"' "$TARGET" > /tmp/target_topics.txt

echo "Top 10 topics by message count:"
echo ""
printf "%-40s %15s %15s %10s\n" "Topic" "Source Msgs" "Target Msgs" "Match %"
echo "------------------------------------------------------------------------------------"

join -t'|' /tmp/source_topics.txt /tmp/target_topics.txt | \
    awk -F'|' '{
        diff = $5 - $3
        percent = ($3 > 0) ? ($5 / $3) * 100 : 0
        printf "%-40s %15s %15s %9.2f%%\n", substr($1, 1, 40), $3, $5, percent
    }' | sort -t' ' -k3 -nr | head -10

# Missing topics
echo ""
echo "📋 Missing Topics:"
echo "------------------"

comm -23 <(jq -r '.topics[].name' "$SOURCE" | sort) <(jq -r '.topics[].name' "$TARGET" | sort) > /tmp/missing_topics.txt
MISSING_COUNT=$(wc -l < /tmp/missing_topics.txt)

if [ "$MISSING_COUNT" -eq 0 ]; then
    echo "✅ No missing topics"
else
    echo "⚠️  $MISSING_COUNT topics missing in target:"
    head -20 /tmp/missing_topics.txt
    if [ "$MISSING_COUNT" -gt 20 ]; then
        echo "... and $((MISSING_COUNT - 20)) more"
    fi
fi

# Extra topics
echo ""
echo "📋 Extra Topics:"
echo "----------------"

comm -13 <(jq -r '.topics[].name' "$SOURCE" | sort) <(jq -r '.topics[].name' "$TARGET" | sort) > /tmp/extra_topics.txt
EXTRA_COUNT=$(wc -l < /tmp/extra_topics.txt)

if [ "$EXTRA_COUNT" -eq 0 ]; then
    echo "✅ No extra topics"
else
    echo "ℹ️  $EXTRA_COUNT topics only in target:"
    head -20 /tmp/extra_topics.txt
    if [ "$EXTRA_COUNT" -gt 20 ]; then
        echo "... and $((EXTRA_COUNT - 20)) more"
    fi
fi

echo ""
echo "=========================================="
echo "Migration Validation Summary:"
echo "=========================================="

if [ "$TOPIC_DIFF" -eq 0 ] && (( $(echo "$MESSAGE_PERCENT >= 99.0" | bc -l) )); then
    echo "✅ Migration appears SUCCESSFUL"
    echo "   - All topics present"
    echo "   - Message counts match (>99%)"
elif [ "$MISSING_COUNT" -gt 0 ]; then
    echo "⚠️  Migration INCOMPLETE"
    echo "   - $MISSING_COUNT topics missing"
    echo "   - Review migration process"
else
    echo "ℹ️  Migration IN PROGRESS or PARTIAL"
    echo "   - Topics match, but message counts differ"
    echo "   - Producers may still be active"
fi

echo ""

# Cleanup
rm -f /tmp/source_topics.txt /tmp/target_topics.txt /tmp/missing_topics.txt /tmp/extra_topics.txt
