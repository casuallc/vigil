## 🧪 RabbitMQ 交换器路由规则集成测试用例

> **测试前提**：
> - 使用 AMQP 0-9-1 协议
> - RabbitMQ 服务器正常运行
> - 每个测试使用独立连接和通道
> - 测试完成后自动清理创建的资源

---

### ✅ 分类一：Direct Exchange 路由

| ID | 描述 | Exchange | Type | Routing Key | Queue | Binding Key | Expected Result |
|----|------|----------|------|-------------|-------|-------------|----------------|
| RB-ROUT-01 | 精确匹配路由 | `test-direct` | `direct` | `key1` | `queue1` | `key1` | ✅ 消息路由到 `queue1` |
| RB-ROUT-02 | 不匹配路由 | `test-direct` | `direct` | `key2` | `queue1` | `key1` | ❌ 消息不路由到 `queue1` |
| RB-ROUT-03 | 多队列匹配相同键 | `test-direct` | `direct` | `shared-key` | `queue1`, `queue2` | `shared-key` | ✅ 消息路由到 `queue1` 和 `queue2` |

---

### ✅ 分类二：Topic Exchange 路由

| ID | 描述 | Exchange | Type | Routing Key | Queue | Binding Key | Expected Result |
|----|------|----------|------|-------------|-------|-------------|----------------|
| RB-ROUT-04 | 精确主题匹配 | `test-topic` | `topic` | `sensor.temp.room1` | `temp-queue` | `sensor.temp.room1` | ✅ 消息路由到 `temp-queue` |
| RB-ROUT-05 | 单层通配符匹配 | `test-topic` | `topic` | `sensor.temp.room2` | `temp-queue` | `sensor.temp.*` | ✅ 消息路由到 `temp-queue` |
| RB-ROUT-06 | 多层通配符匹配 | `test-topic` | `topic` | `sensor.temp.room1.floor2` | `all-sensors` | `sensor.#` | ✅ 消息路由到 `all-sensors` |
| RB-ROUT-07 | 多层通配符不匹配 | `test-topic` | `topic` | `sensor.temp.room1` | `other-sensors` | `other.#` | ❌ 消息不路由到 `other-sensors` |
| RB-ROUT-08 | 前缀通配符匹配 | `test-topic` | `topic` | `sensor.humid.room1` | `all-sensors` | `sensor.*.room1` | ✅ 消息路由到 `all-sensors` |

---

### ✅ 分类三：Fanout Exchange 路由

| ID | 描述 | Exchange | Type | Routing Key | Queue | Binding Key | Expected Result |
|----|------|----------|------|-------------|-------|-------------|----------------|
| RB-ROUT-09 | 广播到所有绑定队列 | `test-fanout` | `fanout` | `any-key` | `queue1`, `queue2`, `queue3` | `any-binding` | ✅ 消息路由到所有三个队列 |
| RB-ROUT-10 | 忽略路由键 | `test-fanout` | `fanout` | ``（空） | `queue1` | `binding-key` | ✅ 消息路由到 `queue1` |
| RB-ROUT-11 | 新绑定队列接收消息 | `test-fanout` | `fanout` | `broadcast` | `queue-new` | `any-key` | ✅ 新绑定队列能接收消息 |

---

### ✅ 分类四：Headers Exchange 路由

| ID | 描述 | Exchange | Type | Headers | Queue | Match Type | Binding Headers | Expected Result |
|----|------|----------|------|---------|-------|------------|----------------|----------------|
| RB-ROUT-12 | 精确匹配所有 headers | `test-headers` | `headers` | `{"type":"temp","room":"1"}` | `headers-queue` | `all` | `{"type":"temp","room":"1"}` | ✅ 消息路由到 `headers-queue` |
| RB-ROUT-13 | 匹配任意 headers | `test-headers` | `headers` | `{"type":"humid","room":"2"}` | `any-headers-queue` | `any` | `{"type":"temp","room":"2"}` | ✅ 消息路由到 `any-headers-queue` |
| RB-ROUT-14 | 不匹配 headers | `test-headers` | `headers` | `{"type":"pressure"}` | `headers-queue` | `all` | `{"type":"temp"}` | ❌ 消息不路由到 `headers-queue` |

---

## ✅ 总结：覆盖维度

| 维度 | 是否覆盖 |
|------|--------|
| Direct Exchange 精确匹配 | ✅ |
| Topic Exchange 通配符匹配 | ✅ |
| Fanout Exchange 广播 | ✅ |
| Headers Exchange 头匹配 | ✅ |
| 多队列绑定 | ✅ |
| 路由失败场景 | ✅ |
| 动态绑定 | ✅ |

---