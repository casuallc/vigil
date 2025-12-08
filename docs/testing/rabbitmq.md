以下是针对 **RabbitMQ Broker**（兼容 AMQP 0-9-1）的 **明确功能点集成测试清单**。每个条目包含：

- **功能点名称**
- **功能描述**
- **测试判断依据（预期行为/验证方式）**

---

### 一、核心功能测试点

#### 1. **消息发布可靠性**
- **描述**：生产者向指定 Exchange 发布消息，消息包含有效路由键（routing key）和 payload。
- **判断依据**：
    - 消息成功写入 RabbitMQ（可通过管理 API 或消费者确认）。
    - 若 Exchange 不存在，发布操作应失败或被拒绝（取决于配置）。
    - 消息属性（如 content-type、delivery_mode、headers）与发送时一致。

#### 2. **Exchange 路由规则验证**
- **描述**：验证不同类型的 Exchange（direct、topic、fanout、headers）是否按预期将消息路由到绑定的 Queue。
- **判断依据**：
    - Direct：仅当 routing key 完全匹配 binding key 时，消息到达对应 Queue。
    - Topic：支持通配符（*、#），消息按 pattern 匹配规则投递。
    - Fanout：忽略 routing key，广播至所有绑定 Queue。
    - Headers：根据 headers 属性（x-match: all/any）决定是否路由。

#### 3. **Queue 绑定（Binding）正确性**
- **描述**：Queue 成功绑定到指定 Exchange，并携带正确的 binding key 或 headers。
- **判断依据**：
    - 绑定后，符合路由规则的消息能进入该 Queue。
    - 解绑后，原应路由至此的消息不再进入。
    - 可通过 RabbitMQ Management API 验证绑定关系存在。

#### 4. **消息消费（Consume）与确认（Ack/Nack）**
- **描述**：消费者从 Queue 中拉取消息，并根据处理结果发送 ack 或 nack。
- **判断依据**：
    - 手动 Ack 模式下，未 Ack 的消息在消费者断开后重新入队（requeue=true）或进入死信队列（requeue=false）。
    - Nack 且 requeue=true 时，消息重新入队并可被再次消费。
    - 消费者成功处理并 Ack 后，消息从 Queue 中移除。

---

### 二、高级功能测试点

#### 5. **死信队列（DLX/DLQ）机制**
- **描述**：当消息被拒绝（nack/reject）、TTL 过期或队列满时，自动转发至死信 Exchange。
- **判断依据**：
    - 配置了 x-dead-letter-exchange 的 Queue，在消息满足死信条件后，消息出现在 DLQ 中。
    - 原 Queue 中该消息消失。
    - 死信消息保留原始消息属性（可选带 x-death header）。

#### 6. **消息 TTL（Time-To-Live）**
- **描述**：消息或 Queue 设置 TTL 后，超时消息被自动丢弃或转入 DLQ。
- **判断依据**：
    - 消息在 Queue 中停留时间超过 TTL 后不再可消费。
    - 若配置 DLX，则超时消息出现在 DLQ。
    - TTL 以毫秒为单位，需验证精度（如 1000ms 后过期）。

#### 7. **消费者并发与公平分发（Fair Dispatch）**
- **描述**：多个消费者订阅同一 Queue 时，消息按 prefetch count 公平分发。
- **判断依据**：
    - 设置 prefetch=1 时，未 Ack 的消费者不再接收新消息。
    - 多个消费者负载大致均衡（尤其在处理时间较长时）。
    - 新加入的消费者能立即参与消费。

#### 8. **Publisher Confirms（发布确认）**
- **描述**：启用 publisher confirms 后，Broker 对每条消息返回 ack/nack。
- **判断依据**：
    - 消息成功路由到至少一个 Queue 后，收到 confirm ack。
    - 消息因无匹配路由而被丢弃（且 mandatory=false）时，收到 confirm ack（但未入队）。
    - 若 mandatory=true 且无路由，收到 basic.return + confirm nack。
    - 确认顺序与发送顺序一致（若开启有序确认）。

---

### 三、扩展功能测试点

#### 9. **Mandatory 标志行为**
- **描述**：发布消息时设置 mandatory=true，若无法路由则返回给生产者。
- **判断依据**：
    - 当消息无法路由到任何 Queue 时，生产者收到 basic.return。
    - 若有至少一个匹配 Queue，则正常入队且无 return。
    - mandatory=false 时，无法路由的消息静默丢弃（无 return）。

#### 10. **Alternate Exchange（备用交换器）**
- **描述**：当消息无法路由到任何队列时，若 Exchange 配置了 alternate-exchange 参数，则消息被转发至该备用 Exchange。
- **判断依据**：
    - 发布一条 routing key 无匹配 binding 的消息到配置了 alternate-exchange 的 Exchange。
    - 消息出现在备用 Exchange 所绑定的 Queue 中。
    - 若未配置 alternate-exchange，消息被丢弃（无 DLQ 或 return 时）。

#### 11. **延迟队列（通过插件 rabbitmq-delayed-message-exchange）**
- **描述**：使用 x-delayed-message 类型 Exchange，消息在指定延迟时间后才被投递到绑定的 Queue。
- **判断依据**：
    - 发布消息时设置 header x-delay（单位毫秒）。
    - 消息在设定延迟时间后才出现在目标 Queue。
    - 延迟期间消息不可被消费。
    - 插件未启用时，创建该类型 Exchange 应失败。

#### 12. **Message Redelivery Count（重投递次数追踪）**
- **描述**：消息被 reject/nack 且 requeue=true 后，可通过 x-redelivered 标识是否为重投递。
- **判断依据**：
    - 首次消费时，redelivered = false。
    - 拒绝并 requeue 后再次消费，redelivered = true。
    - 可结合此标志实现最大重试次数控制。

#### 13. **Per-Queue TTL 与 Message TTL 的优先级关系**
- **描述**：Queue 设置 x-message-ttl，消息自身也设置 TTL，以较小值为准。
- **判断依据**：
    - Queue TTL=5000ms，消息 TTL=2000ms → 消息 2 秒后过期。
    - Queue TTL=2000ms，消息 TTL=5000ms → 消息 2 秒后过期。
    - 过期后行为符合 DLQ 或丢弃规则。

#### 14. **Exclusive Queue 生命周期**
- **描述**：声明为 exclusive 的 Queue 仅对当前连接可见，连接关闭后自动删除。
- **判断依据**：
    - 同一连接内可消费该 Queue。
    - 其他连接无法看到或绑定该 Queue。
    - 关闭声明该 Queue 的连接后，Queue 立即消失（可通过管理 API 验证）。

#### 15. **Auto-delete Queue 行为**
- **描述**：Queue 设置 auto-delete=true，在最后一个消费者取消后自动删除。
- **判断依据**：
    - 创建 auto-delete Queue 并绑定消费者。
    - 取消所有消费者后，Queue 自动删除。
    - 若从未有消费者，Queue 不会因 auto-delete 而立即删除（需至少一个 consumer 出现过）。

---

> ✅ 以上所有测试点均可通过标准 AMQP 客户端（如 rabbitmq/amqp091-go）配合 RabbitMQ Management API / 日志进行验证。