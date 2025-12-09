## 🧪 RabbitMQ 消费者并发集成测试用例

> **测试前提**：
> - 使用 AMQP 0-9-1 协议
> - RabbitMQ 服务器正常运行
> - 每个测试使用独立连接和通道
> - 测试完成后自动清理创建的资源

---

### ✅ 分类一：基本并发消费

| ID | 描述 | Queue | Consumers | Messages | Expected Result |
|----|------|-------|-----------|----------|----------------|
| RB-CONC-01 | 多个消费者共享队列 | `test-queue` | 2 | 10 | ✅ 消息被均匀分配 |
| RB-CONC-02 | 动态添加消费者 | `test-queue` | 1 → 3 | 10 | ✅ 新消费者参与消费 |
| RB-CONC-03 | 消费者断开连接 | `test-queue` | 2 → 1 | 10 | ✅ 剩余消费者处理所有消息 |
| RB-CONC-04 | 大量消费者 | `test-queue` | 10 | 100 | ✅ 所有消费者参与消费 |

---

### ✅ 分类二：公平分发

| ID | 描述 | Queue | Prefetch | Consumers | Expected Result |
|----|------|-------|----------|-----------|----------------|
| RB-CONC-05 | 公平分发配置 | `test-queue` | `prefetch: 1` | 2 | ✅ 每条消息确认后才分配新消息 |
| RB-CONC-06 | 预取多条消息 | `test-queue` | `prefetch: 5` | 2 | ✅ 每个消费者预取 5 条消息 |
| RB-CONC-07 | 无预取限制 | `test-queue` | `prefetch: 0` | 2 | ✅ 消息立即全部分配 |
| RB-CONC-08 | 不同预取值 | `test-queue` | `prefetch: 1, 5` | 2 | ✅ 预取 5 的消费者处理更多消息 |

---

### ✅ 分类三：并发消费场景

| ID | 描述 | Queue | Consumer Behavior | Expected Result |
|----|------|-------|-----------------|----------------|
| RB-CONC-09 | 消费者处理速度不同 | `test-queue` | `sleep(100ms, 500ms)` | ✅ 公平分发确保负载均衡 |
| RB-CONC-10 | 部分消费者失败 | `test-queue` | `fail on some messages` | ✅ 其他消费者继续处理 |
| RB-CONC-11 | 消费者重启 | `test-queue` | `restart consumer` | ✅ 消息不丢失，重新分配 |
| RB-CONC-12 | 高并发消息 | `test-queue` | 1000 messages | ✅ 所有消息被成功消费 |

---

### ✅ 分类四：消费者属性

| ID | 描述 | Queue | Consumer Args | Expected Result |
|----|------|-------|--------------|----------------|
| RB-CONC-13 | 消费者优先级 | `test-queue` | `x-priority: 1, 5` | ✅ 优先级 5 的消费者优先获得消息 |
| RB-CONC-14 | 排他性消费者 | `test-queue` | `exclusive: true` | ✅ 只有一个消费者能连接 |
| RB-CONC-15 | 消费者标签 | `test-queue` | `consumer_tag: custom-tag` | ✅ 消费者使用指定标签 |
| RB-CONC-16 | 消费者取消通知 | `test-queue` | `notify-cancel` | ✅ 收到取消通知 |

---

## ✅ 总结：覆盖维度

| 维度 | 是否覆盖 |
|------|--------|
| 基本并发消费 | ✅ |
| 动态消费者 | ✅ |
| 公平分发 | ✅ |
| 预取配置 | ✅ |
| 消费者优先级 | ✅ |
| 高并发场景 | ✅ |
| 消费者失败处理 | ✅ |
| 消费者属性 | ✅ |

---