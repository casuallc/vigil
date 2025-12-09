以下是遗嘱消息的测试用例。

> 📌 说明：
> - “订阅”列指**监听遗嘱消息的客户端所订阅的主题**
> - “发布”列指**遗嘱消息最终由 Broker 发布的主题**
> - “QoS” 列分别对应订阅和发布的 QoS 级别
> - “预期”描述监听客户端是否应收到消息，以及消息属性

---

### 🧪 MQTT 遗嘱消息（LWT）集成测试用例表

| ID | 描述 | 订阅 | QoS | 发布（遗嘱主题） | QoS | 预期 |
|----|------|------|-----|------------------|-----|------|
| LWT-01 | 客户端异常断开，触发遗嘱 | `clients/status` | 0 | `clients/status` | 0 | ✅ 收到 payload="offline" |
| LWT-02 | 客户端正常 DISCONNECT，不触发遗嘱 | `clients/status` | 0 | `clients/status` | 0 | ❌ 未收到任何消息 |
| LWT-03 | 遗嘱 QoS=1，订阅者 QoS=0 | `status/will` | 0 | `status/will` | 1 | ✅ 收到消息（按订阅 QoS 降级为 0） |
| LWT-04 | 遗嘱 QoS=0，订阅者 QoS=1 | `status/will` | 1 | `status/will` | 0 | ✅ 收到消息（QoS=0） |
| LWT-05 | 遗嘱 retain=true，新订阅者可收到 | `devices/last-will` | 0 | `devices/last-will` | 0 | ✅ 新客户端订阅后立即收到 retained 遗嘱消息 |
| LWT-06 | 遗嘱 retain=false，新订阅者收不到 | `devices/last-will` | 0 | `devices/last-will` | 0 | ❌ 新客户端订阅后无消息 |
| LWT-07 | 多个客户端相同遗嘱主题，独立触发 | `gateway/status` | 0 | `gateway/status` | 0 | ✅ 收到两条独立消息（payload 分别为 "dev1 offline" 和 "dev2 offline"） |
| LWT-08 | 相同 client_id 重连，新遗嘱覆盖旧遗嘱 | `sensor/status` | 0 | `sensor/status` | 0 | ✅ 仅收到新连接设置的遗嘱内容（旧遗嘱被取消） |
| LWT-09 | 遗嘱 payload 为空字符串 | `test/will` | 0 | `test/will` | 0 | ✅ 收到一条 payload 为空（长度为 0）的消息 |
| LWT-10 | 遗嘱主题为空（非法） | — | — | `""`（空） | 0 | ❌ CONNECT 被拒绝（CONNACK 返回非 0，如 MQTT 5.0: 0x82） |
| LWT-11 | Clean Session = true，异常断开仍触发遗嘱 | `session/will` | 0 | `session/will` | 0 | ✅ 收到遗嘱（clean session 不影响 LWT） |
| LWT-12 | Clean Session = false，异常断开仍触发遗嘱 | `session/will` | 0 | `session/will` | 0 | ✅ 收到遗嘱（与 clean session 无关） |
| LWT-13 | 订阅者使用通配符接收遗嘱 | `+/status` | 0 | `device123/status` | 0 | ✅ 收到遗嘱（通配符匹配生效） |
| LWT-14 | 遗嘱主题含特殊字符（合法） | `client/@user/status` | 0 | `client/@user/status` | 0 | ✅ 正常收到消息 |
| LWT-15 | Broker 重启期间客户端掉线（EMQX 行为） | `broker/will` | 0 | `broker/will` | 0 | ❌ 通常**不会**触发遗嘱（会话状态丢失） |
| LWT-16 | 客户端无权限发布遗嘱主题（ACL 限制） | `secret/status` | 0 | `secret/status` | 0 | ⚠️ 连接成功，但异常断开后**静默丢弃**遗嘱（EMQX 默认行为） |

---

### 🔍 补充说明

- **LWT-10**：MQTT 规范明确要求遗嘱主题**必须是非空 UTF-8 字符串**，空主题属于协议错误。
- **LWT-16**：大多数 Broker（包括 EMQX）在 CONNECT 阶段**不校验遗嘱主题的发布权限**，仅在实际发布时检查。若无权限，遗嘱被丢弃，但不会通知客户端。
- **LWT-15**：遗嘱是**运行时会话状态**的一部分，Broker 重启后若未持久化会话，则无法触发。如需高可靠离线通知，建议结合应用层心跳+状态上报。
- **通配符订阅（LWT-13）**：遗嘱消息作为普通 PUBLISH 消息路由，完全支持通配符匹配。

---