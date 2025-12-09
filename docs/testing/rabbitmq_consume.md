## 🧪 RabbitMQ 消息消费集成测试用例

> **测试前提**：
> - 使用 AMQP 0-9-1 协议
> - RabbitMQ 服务器正常运行
> - 每个测试使用独立连接和通道
> - 测试完成后自动清理创建的资源

---

### ✅ 分类一：基本消费操作

| ID | 描述 | Queue | AutoAck | Expected Result |
|----|------|-------|---------|----------------|
| RB-CONS-01 | 成功消费消息 | `test-queue` | `true` | ✅ 消息被消费 |
| RB-CONS-02 | 成功消费并确认 | `test-queue` | `false` | ✅ 消息被消费并确认 |
| RB-CONS-03 | 消费不存在的队列 | `non-existent-queue` | `true` | ❌ 消费失败 |
| RB-CONS-04 | 消费超时 | `empty-queue` | `true` | ✅ 超时返回，无错误 |

---

### ✅ 分类二：消息确认模式

| ID | 描述 | Queue | AutoAck | Expected Result |
|----|------|-------|---------|----------------|
| RB-CONS-05 | 自动确认模式 | `test-queue` | `true` | ✅ 消息自动确认 |
| RB-CONS-06 | 手动确认模式 | `test-queue` | `false` | ✅ 消息需手动确认 |
| RB-CONS-07 | 批量确认消息 | `test-queue` | `false` | ✅ 多条消息可批量确认 |
| RB-CONS-08 | 拒绝消息并重新入队 | `test-queue` | `false` | ✅ 消息被拒绝并重新入队 |

---

### ✅ 分类三：消息处理

| ID | 描述 | Queue | Handler Behavior | Expected Result |
|----|------|-------|-----------------|----------------|
| RB-CONS-09 | 成功处理消息 | `test-queue` | `msg.Ack()` | ✅ 消息被成功处理 |
| RB-CONS-10 | 处理失败拒绝消息 | `test-queue` | `msg.Nack(false, false)` | ✅ 消息被拒绝，不重新入队 |
| RB-CONS-11 | 处理失败重新入队 | `test-queue` | `msg.Nack(false, true)` | ✅ 消息被拒绝，重新入队 |
| RB-CONS-12 | 处理超时 | `test-queue` | `sleep(6s)` | ✅ 消费者超时，消息可被其他消费者消费 |

---

### ✅ 分类四：消费者属性

| ID | 描述 | Queue | Consumer Args | Expected Result |
|----|------|-------|--------------|----------------|
| RB-CONS-13 | 带自定义参数的消费者 | `test-queue` | `{"x-priority":1}` | ✅ 消费者成功启动 |
| RB-CONS-14 | 排他性消费者 | `test-queue` | `exclusive: true` | ✅ 只有一个排他消费者能消费 |
| RB-CONS-15 | 消费者标签 | `test-queue` | `consumer_tag: "test-consumer"` | ✅ 消费者使用指定标签 |
| RB-CONS-16 | 消费者取消 | `test-queue` | `cancel` | ✅ 消费者被成功取消 |

---

## ✅ 总结：覆盖维度

| 维度 | 是否覆盖 |
|------|--------|
| 基本消费操作 | ✅ |
| 自动确认模式 | ✅ |
| 手动确认模式 | ✅ |
| 消息拒绝与重新入队 | ✅ |
| 消费者属性 | ✅ |
| 消费超时 | ✅ |
| 消费者取消 | ✅ |
| 批量确认 | ✅ |

---