## 🧪 MQTT Keep Alive 集成测试用例

> **测试前提**：
> - 使用 MQTT 3.1.1 或 5.0 协议
> - Broker：EMQX（行为符合规范）
> - 每个测试使用独立客户端（clean session = true）
> - 关闭自动重连功能

---

### ✅ 分类一：基本 Keep Alive 功能验证

| ID | 描述 | 客户端配置 | 测试步骤 | 预期结果 | 说明 |
|----|------|----------|--------|--------|------|
| KA-01 | Keep Alive 正常工作 | Keep Alive=60s<br>Clean Start=true | 1. 客户端连接<br>2. 每 50s 发送 PINGREQ<br>3. 保持 5 分钟 | ✅ 连接保持正常<br>✅ 收到 PINGRESP | 正常心跳机制 |
| KA-02 | 超过 Keep Alive 时间未发送 PINGREQ | Keep Alive=10s<br>Clean Start=true | 1. 客户端连接<br>2. 不发送 PINGREQ<br>3. 等待 15s | ❌ 连接被 Broker 断开 | Broker 在 1.5×Keep Alive 后断开连接 |
| KA-03 | Keep Alive=0（禁用） | Keep Alive=0<br>Clean Start=true | 1. 客户端连接，设置 Keep Alive=0<br>2. 保持 2 分钟不发送任何消息 | ✅ 连接保持正常 | Keep Alive=0 表示禁用心跳 |

---

### ✅ 分类二：Keep Alive 与连接状态

| ID | 描述 | 客户端配置 | 测试步骤 | 预期结果 |
|----|------|----------|--------|--------|
| KA-04 | 网络断开触发 Keep Alive 超时 | Keep Alive=10s | 1. 客户端连接<br>2. 模拟网络断开（如关闭网卡）<br>3. 等待 15s | ❌ 连接被 Broker 断开 |
| KA-05 | 恢复网络后重连 | Keep Alive=10s<br>Auto Reconnect=true | 1. 客户端连接<br>2. 断开网络 15s<br>3. 恢复网络 | ✅ 自动重连成功 |
| KA-06 | 频繁 PINGREQ 不影响连接 | Keep Alive=60s | 1. 客户端连接<br>2. 每 1s 发送 PINGREQ<br>3. 持续 1 分钟 | ✅ 连接保持正常 | Broker 应正常处理频繁心跳 |

---

### ✅ 分类三：Keep Alive 与 QoS 组合

| ID | 描述 | 客户端配置 | 测试步骤 | 预期结果 |
|----|------|----------|--------|--------|
| KA-07 | 发送消息代替 PINGREQ | Keep Alive=10s | 1. 客户端连接<br>2. 每 8s 发送一条 QoS 0 消息<br>3. 持续 1 分钟 | ✅ 连接保持正常 | 正常 PUBLISH 消息可代替 PINGREQ |
| KA-08 | 接收消息不代替 PINGREQ | Keep Alive=10s | 1. 客户端连接<br>2. 只接收消息，不发送任何消息<br>3. 等待 15s | ❌ 连接被 Broker 断开 | 只有发送的消息能重置 Keep Alive 计时器 |

---

### ✅ 分类四：边界条件与特殊情况

| ID | 描述 | 客户端配置 | 测试步骤 | 预期结果 | 说明 |
|----|------|----------|--------|--------|------|
| KA-09 | 极小 Keep Alive 值 | Keep Alive=1s | 1. 客户端连接<br>2. 每 0.5s 发送 PINGREQ<br>3. 持续 30s | ✅ 连接保持正常 | 验证 Broker 处理小 Keep Alive 值的能力 |
| KA-10 | 极大 Keep Alive 值 | Keep Alive=3600s (1h) | 1. 客户端连接<br>2. 保持 5 分钟不发送消息<br>3. 发送 PINGREQ | ✅ 连接保持正常<br>✅ 收到 PINGRESP | 验证长时间 Keep Alive 的支持 |
| KA-11 | 客户端主动断开连接 | Keep Alive=60s | 1. 客户端连接<br>2. 主动发送 DISCONNECT<br>3. 等待 2s | ✅ 连接正常关闭<br>✅ Broker 无异常 | 正常关闭不依赖 Keep Alive |

---

### ✅ 分类五：MQTT 5.0 Keep Alive 扩展

| ID | 描述 | 客户端配置 | 测试步骤 | 预期结果 |
|----|------|----------|--------|--------|
| KA-12 | MQTT 5.0 Keep Alive | Keep Alive=30s<br>MQTT 5.0 | 1. 客户端连接<br>2. 超过时间未发送 PINGREQ<br>3. 等待 45s | ❌ 连接被断开<br>✅ 收到 Disconnect Reason Code=0x8D (Keep Alive Timeout) | MQTT 5.0 提供断开原因码 |
| KA-13 | Session Expiry 与 Keep Alive | Keep Alive=10s<br>Session Expiry=30s | 1. 客户端连接，Clean Start=false<br>2. 断开网络 15s<br>3. 10s 后重连 | ✅ 重连成功<br>✅ 会话保持 | Session Expiry > Keep Alive 时会话可恢复 |

---

## ✅ 总结：覆盖维度

| 维度 | 是否覆盖 |
|------|--------|
| 基本 Keep Alive 机制 | ✅ |
| Keep Alive 超时断开 | ✅ |
| Keep Alive=0（禁用） | ✅ |
| 网络断开场景 | ✅ |
| 自动重连结合 | ✅ |
| 消息代替 PINGREQ | ✅ |
| 边界值测试 | ✅ |
| MQTT 5.0 扩展 | ✅ |
| 会话保持结合 | ✅ |

---