## 🧪 RabbitMQ 死信队列集成测试用例

> **测试前提**：
> - 使用 AMQP 0-9-1 协议
> - RabbitMQ 服务器正常运行
> - 每个测试使用独立连接和通道
> - 测试完成后自动清理创建的资源

---

### ✅ 分类一：DLQ 基本功能

| ID | 描述 | Main Queue | DLQ | DLX | Expected Result |
|----|------|------------|-----|-----|----------------|
| RB-DLQ-01 | 成功配置死信队列 | `main-queue` | `dlq-queue` | `dlx-exchange` | ✅ 死信队列配置成功 |
| RB-DLQ-02 | 消息进入死信队列 | `main-queue` | `dlq-queue` | `dlx-exchange` | ✅ 拒绝的消息进入 DLQ |
| RB-DLQ-03 | 从 DLQ 消费消息 | `main-queue` | `dlq-queue` | `dlx-exchange` | ✅ 能从 DLQ 消费消息 |

---

### ✅ 分类二：触发死信的场景

| ID | 描述 | Main Queue | DLQ | Trigger Condition | Expected Result |
|----|------|------------|-----|-------------------|----------------|
| RB-DLQ-04 | 消息被拒绝 | `main-queue` | `dlq-queue` | `msg.Nack(false, false)` | ✅ 消息进入 DLQ |
| RB-DLQ-05 | 消息 TTL 过期 | `main-queue` | `dlq-queue` | `x-message-ttl: 1000` | ✅ 消息过期后进入 DLQ |
| RB-DLQ-06 | 队列达到最大长度 | `main-queue` | `dlq-queue` | `x-max-length: 1` | ✅ 新消息进入 DLQ |
| RB-DLQ-07 | 消费者断开连接 | `main-queue` | `dlq-queue` | 消费者断开，`no-ack` | ✅ 未确认消息进入 DLQ |

---

### ✅ 分类三：DLQ 配置

| ID | 描述 | Main Queue Args | DLX | DLQ | Expected Result |
|----|------|----------------|-----|-----|----------------|
| RB-DLQ-08 | 配置死信交换器 | `x-dead-letter-exchange: dlx` | `dlx` | `dlq` | ✅ 消息路由到 DLX |
| RB-DLQ-09 | 配置死信路由键 | `x-dead-letter-routing-key: dl-key` | `dlx` | `dlq` | ✅ 消息使用指定路由键进入 DLQ |
| RB-DLQ-10 | 配置死信消息TTL | `x-dead-letter-exchange: dlx`, `x-message-ttl: 1000` | `dlx` | `dlq` | ✅ 消息过期后进入 DLQ |
| RB-DLQ-11 | 配置最大长度 | `x-dead-letter-exchange: dlx`, `x-max-length: 2` | `dlx` | `dlq` | ✅ 超过长度的消息进入 DLQ |

---

### ✅ 分类四：DLQ 高级特性

| ID | 描述 | Main Queue | DLQ | DLX | Expected Result |
|----|------|------------|-----|-----|----------------|
| RB-DLQ-12 | DLQ 消息保留原始属性 | `main-queue` | `dlq-queue` | `dlx-exchange` | ✅ DLQ 消息保留原始属性 |
| RB-DLQ-13 | 多层 DLQ | `main-queue` | `dlq1`, `dlq2` | `dlx1`, `dlx2` | ✅ 消息可依次进入多级 DLQ |
| RB-DLQ-14 | DLQ 优先级队列 | `main-queue` | `dlq-priority` | `dlx-exchange` | ✅ DLQ 支持优先级 |

---

## ✅ 总结：覆盖维度

| 维度 | 是否覆盖 |
|------|--------|
| DLQ 基本配置 | ✅ |
| 死信触发场景 | ✅ |
| 死信原因类型 | ✅ |
| DLQ 消息消费 | ✅ |
| 高级 DLQ 配置 | ✅ |
| 多层 DLQ | ✅ |
| DLQ 优先级 | ✅ |

---