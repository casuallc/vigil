## 🧪 RabbitMQ 消息发布集成测试用例

> **测试前提**：
> - 使用 AMQP 0-9-1 协议
> - RabbitMQ 服务器正常运行
> - 已创建测试 Exchange：`test-exchange`（direct 类型）
> - 每个测试使用独立连接和通道

---

### ✅ 分类一：基本发布功能

| ID | 描述 | Exchange | Routing Key | Message | Expected Result | 说明 |
|----|------|----------|-------------|---------|----------------|------|
| RB-PUB-01 | 基本消息发布 | `test-exchange` | `test.key` | `Hello World` | ✅ 消息成功发布（无错误） | 基本发布功能验证 |
| RB-PUB-02 | 发布到默认 Exchange | `(default)` | `test.queue` | `Default Exchange Test` | ✅ 消息成功发布到队列 `test.queue` | 默认 Exchange 直接路由到同名队列 |
| RB-PUB-03 | 发布到不存在的 Exchange | `non-existent-exchange` | `test.key` | `Test Message` | ❌ 发布失败（通道关闭或返回错误） | Exchange 不存在时应报错 |

---

### ✅ 分类二：消息属性

| ID | 描述 | Exchange | Routing Key | Message Properties | Expected Result |
|----|------|----------|-------------|--------------------|----------------|
| RB-PUB-04 | 持久化消息发布 | `test-exchange` | `test.key` | `delivery_mode=2` | ✅ 消息被持久化 |
| RB-PUB-05 | 非持久化消息发布 | `test-exchange` | `test.key` | `delivery_mode=1` | ✅ 消息未被持久化 |
| RB-PUB-06 | 带自定义 headers | `test-exchange` | `test.key` | `headers={"type":"test","priority":1}` | ✅ 消息携带正确 headers |
| RB-PUB-07 | 带 content-type | `test-exchange` | `test.key` | `content_type="application/json"` | ✅ 消息 content-type 正确 |

---

### ✅ 分类三：发布确认

| ID | 描述 | Exchange | Routing Key | Confirm Mode | Expected Result |
|----|------|----------|-------------|--------------|----------------|
| RB-PUB-08 | 发布确认：成功路由 | `test-exchange` | `test.key` | `confirm_mode=true` | ✅ 收到发布确认（ack） |
| RB-PUB-09 | 发布确认：路由失败 | `test-exchange` | `non-existent-key` | `confirm_mode=true` | ✅ 收到发布确认（ack，即使路由失败） |
| RB-PUB-10 | 发布确认：mandatory=true | `test-exchange` | `non-existent-key` | `confirm_mode=true, mandatory=true` | ✅ 收到 basic.return + 发布确认 |

---

### ✅ 分类四：高并发发布

| ID | 描述 | Exchange | Routing Key | Concurrency | Expected Result |
|----|------|----------|-------------|-------------|----------------|
| RB-PUB-11 | 高并发发布 | `test-exchange` | `test.key` | `100 并发发布者，10,000 条消息` | ✅ 所有消息成功发布 |
| RB-PUB-12 | 快速连续发布 | `test-exchange` | `test.key` | `每 1ms 发布一条，1,000 条` | ✅ 所有消息成功发布 |

---

## ✅ 总结：覆盖维度

| 维度 | 是否覆盖 |
|------|--------|
| 基本发布功能 | ✅ |
| 默认 Exchange 发布 | ✅ |
| 不存在 Exchange 处理 | ✅ |
| 持久化与非持久化消息 | ✅ |
| 自定义消息属性 | ✅ |
| 发布确认机制 | ✅ |
| 高并发发布 | ✅ |
| mandatory 标志 | ✅ |

---