## 🧪 MQTT 保留消息（Retained Messages）集成测试用例

> **测试前提**：
> - 使用 MQTT 3.1.1 或 5.0 协议
> - Broker：EMQX（行为符合规范）
> - 每个测试使用独立客户端（clean session = true）
> - 所有测试使用 QoS 0（除非特别说明）

---

### ✅ 分类一：基本保留消息行为

| ID | 描述 | 订阅主题 | 发布操作 | 预期结果 | 说明 |
|----|------|--------|--------|--------|------|
| RET-01 | 新订阅者收到保留消息 | `sensor/status` | 1. 发布 `sensor/status` payload="online" retain=true<br>2. 新客户端订阅 | ✅ 订阅后立即收到 payload="online" | 基本保留消息功能 |
| RET-02 | 发布空 payload 清除保留消息 | `sensor/status` | 1. 发布 `sensor/status` payload="online" retain=true<br>2. 发布 `sensor/status` payload="" retain=true<br>3. 新客户端订阅 | ❌ 未收到任何保留消息 | 空 payload + retain=true 清除保留消息 |
| RET-03 | 非 retain 消息不影响保留消息 | `sensor/status` | 1. 发布 `sensor/status` payload="online" retain=true<br>2. 发布 `sensor/status` payload="offline" retain=false<br>3. 新客户端订阅 | ✅ 收到 payload="online" | 非 retain 消息不替换保留消息 |
| RET-04 | 新 retain 消息替换旧保留消息 | `sensor/status` | 1. 发布 `sensor/status` payload="online" retain=true<br>2. 发布 `sensor/status` payload="offline" retain=true<br>3. 新客户端订阅 | ✅ 收到 payload="offline" | 新 retain 消息覆盖旧消息 |

---

### ✅ 分类二：保留消息与 QoS 组合

| ID | 描述 | 订阅主题 | 订阅 QoS | 发布操作 | 发布 QoS | 预期结果 |
|----|------|--------|--------|--------|--------|--------|
| RET-05 | 发布 QoS 1 + retain | `sensor/status` | 0 | 发布 `sensor/status` payload="online" retain=true QoS=1 | ✅ 订阅后收到 payload="online" | 保留消息支持 QoS 1 |
| RET-06 | 发布 QoS 2 + retain | `sensor/status` | 1 | 发布 `sensor/status` payload="online" retain=true QoS=2 | ✅ 订阅后收到 payload="online" QoS=1 | 按订阅 QoS 降级 |
| RET-07 | 订阅 QoS 1 接收 retain 消息 | `sensor/status` | 1 | 发布 `sensor/status` payload="online" retain=true QoS=0 | ✅ 收到 QoS=0 | 保留消息按实际发布 QoS 传递 |

---

### ✅ 分类三：保留消息与通配符订阅

| ID | 描述 | 订阅主题 | 发布操作 | 预期结果 |
|----|------|--------|--------|--------|
| RET-08 | 单层通配符接收保留消息 | `sensor/+/status` | 发布 `sensor/room1/status` payload="online" retain=true | ✅ 订阅后收到 payload="online" |
| RET-09 | 多层通配符接收保留消息 | `sensor/#` | 发布 `sensor/room1/status` payload="online" retain=true | ✅ 订阅后收到 payload="online" |
| RET-10 | 通配符订阅匹配多个保留消息 | `sensor/#` | 1. 发布 `sensor/room1/status` payload="online1" retain=true<br>2. 发布 `sensor/room2/status` payload="online2" retain=true | ✅ 订阅后收到两条保留消息 |

---

### ✅ 分类四：边界条件与特殊情况

| ID | 描述 | 订阅主题 | 发布操作 | 预期结果 | 说明 |
|----|------|--------|--------|--------|------|
| RET-11 | 保留消息与 LWT 结合 | `device/status` | 1. 客户端 A 连接，设置 LWT `device/status` payload="offline" retain=true<br>2. 客户端 A 异常断开<br>3. 新客户端订阅 | ✅ 收到 payload="offline" | LWT 支持 retain |
| RET-12 | 大量保留消息处理 | `test/topic/` (100+ topics) | 发布 100+ 不同主题的 retain 消息 | ✅ 新订阅者能正确收到对应主题的保留消息 | 验证 Broker 处理能力 |
| RET-13 | 重复订阅不影响保留消息 | `sensor/status` | 1. 发布 retain 消息<br>2. 客户端多次订阅同一主题 | ✅ 仅首次订阅收到保留消息 | 后续订阅不再重复发送 |

---

### ✅ 分类五：MQTT 5.0 保留消息扩展

| ID | 描述 | 订阅主题 | 订阅配置 | 发布操作 | 预期结果 |
|----|------|--------|--------|--------|--------|
| RET-14 | Retain Handling=0 | `sensor/status` | Retain Handling=0 | 已有保留消息 | ✅ 收到保留消息 |
| RET-15 | Retain Handling=1 | `sensor/status` | Retain Handling=1 | 已有保留消息 | ❌ 仅当当前主题无会话存在时才收到 |
| RET-16 | Retain Handling=2 | `sensor/status` | Retain Handling=2 | 已有保留消息 | ❌ 始终不收到保留消息 |

---

## ✅ 总结：覆盖维度

| 维度 | 是否覆盖 |
|------|--------|
| 基本保留消息发布与接收 | ✅ |
| 保留消息清除机制 | ✅ |
| 保留消息与 QoS 组合 | ✅ |
| 保留消息与通配符订阅 | ✅ |
| 保留消息与 LWT 结合 | ✅ |
| MQTT 5.0 Retain Handling | ✅ |
| 边界条件测试 | ✅ |

---