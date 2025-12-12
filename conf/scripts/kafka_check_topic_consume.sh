#!/bin/bash

# === 配置 ===
TOPIC=${TOPIC:-"topic01"}
BOOTSTRAP_SERVER=${BOOTSTRAP_SERVER:-"127.0.0.1:9092"}
KAFKA_DIR=${KAFKA_DIR:-"/opt/kafka"}
# ============

cd $KAFKA_DIR

echo "监控 Topic [$TOPIC] 的消费状态（最新位置 / 客户端 / 堆积）\n"

ALL_GROUPS=$(bin/kafka-consumer-groups.sh --bootstrap-server "$BOOTSTRAP_SERVER" --list 2>/dev/null)

if [ -z "$ALL_GROUPS" ]; then
  echo "❌ 无法连接 Kafka 或无消费者组。"
  exit 1
fi

while IFS= read -r GROUP; do
  [ -z "$GROUP" ] && continue

  OUTPUT=$(bin/kafka-consumer-groups.sh --bootstrap-server "$BOOTSTRAP_SERVER" --describe --group "$GROUP" 2>/dev/null)

  # 提取该 Topic 的分区行
  TOPIC_LINES=$(echo "$OUTPUT" | grep "^$TOPIC[[:space:]]")
  if [ -n "$TOPIC_LINES" ]; then
    echo "──────────────────────────────────────"
    echo "消费者组: $GROUP"
    echo "──────────────────────────────────────"

    while IFS= read -r LINE; do
      PARTITION=$(echo "$LINE" | awk '{print $2}')
      CURRENT=$(echo "$LINE" | awk '{print $3}')
      END=$(echo "$LINE" | awk '{print $4}')
      LAG=$(echo "$LINE" | awk '{print $5}')
      CONSUMER_ID=$(echo "$LINE" | awk '{print $6}')
      HOST=$(echo "$LINE" | awk '{print $7}')
      CLIENT_ID=$(echo "$LINE" | awk '{print $8}')

      # 判断是否堆积且无消费者
      ALERT=""
      if [ "$LAG" -gt 0 ] && [ "$CONSUMER_ID" = "-" ]; then
        ALERT="[堆积且无活跃消费者！]"
      elif [ "$LAG" -gt 1000 ]; then
        ALERT="[高堆积！]"
      fi

      printf "分区: %2s | 最新位置: %8s | 已消费: %8s | 堆积: %6s | 客户端: %-15s | Host: %-15s %s\n" \
             "$PARTITION" "$END" "$CURRENT" "$LAG" "$CLIENT_ID" "$HOST" "$ALERT"
    done <<< "$TOPIC_LINES"
    echo ""
  fi
done <<< "$ALL_GROUPS"