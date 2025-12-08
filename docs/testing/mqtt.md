以下是针对 **EMQX Broker**（以 EMQX 5.x 为主，兼容 MQTT 3.1.1 和 5.0）的 **明确功能点集成测试清单**。每个条目包含：

- **功能点名称**
- **功能描述**
- **测试判断依据（预期行为/验证方式）**

---

### 一、MQTT 3.1.1 功能测试点（EMQX 支持）

#### 1. **Client Connect with Clean Session**
- **描述**：客户端使用 cleanSession=true 或 false 连接。
- **判断依据**：
    - cleanSession=true：连接后无历史订阅，离线消息不保留。
    - cleanSession=false：重连后恢复原订阅；QoS>0 的离线消息在重连后投递。

#### 2. **QoS 0/1/2 消息投递**
- **描述**：发布不同 QoS 级别的消息，验证投递语义。
- **判断依据**：
    - QoS 0：接收方收到一次或未收到（不可靠）。
    - QoS 1：接收方至少收到一次（可能重复，需去重逻辑验证）。
    - QoS 2：接收方恰好收到一次（EMQX 默认支持 QoS 2）。

#### 3. **Retained Message 发布与获取**
- **描述**：发布 retained=true 消息，新订阅者是否立即收到。
- **判断依据**：
    - 新订阅者订阅该 topic 后，**立即收到最后一条 retained 消息**。
    - 发布空 payload + retained=true 后，后续新订阅者**不再收到 retained 消息**。

#### 4. **通配符订阅匹配**
- **描述**：使用 `+` 和 `#` 订阅，验证消息路由。
- **判断依据**：
    - `sensor/+/temp` 匹配 `sensor/room1/temp`，但不匹配 `sensor/room1/humid/temp`。
    - `sensor/#` 匹配 `sensor/a`、`sensor/a/b/c`，但不匹配 `sensors/a`。
    - 不合法通配符（如 `sensor/#/extra`）应被拒绝（EMQX 返回 SUBACK 失败）。

#### 5. **Keep Alive 超时断连**
- **描述**：客户端设置 keep alive，超时未通信应被断开。
- **判断依据**：
    - 客户端在 1.5×keep_alive 时间内未发任何包，**EMQX 主动关闭 TCP 连接**。
    - 客户端定期发 PINGREQ，EMQX 回 PINGRESP，连接保持。

#### 6. **ACL 控制（基于内置 ACL 或文件）**
- **描述**：用户对 topic 的发布/订阅权限受控。
- **判断依据**：
    - 无权限用户尝试订阅/发布受限 topic，**操作被拒绝（无消息到达 / SUBACK=0x80）**。
    - 有权限用户可正常收发。

#### 7. **TLS 加密连接（端口 8883）**
- **描述**：客户端通过 TLS 连接 EMQX。
- **判断依据**：
    - 使用有效证书可成功 CONNECT。
    - 无证书或证书错误时连接失败（TCP 层或协议层拒绝）。

---

### 二、MQTT 5.0 特有功能测试点（EMQX 5.x 支持）

#### 8. **Session Expiry Interval**
- **描述**：CONNECT 时设置 Session Expiry Interval（SEI）。
- **判断依据**：
    - SEI=0：断开后**立即清除会话**（等效 cleanSession=true）。
    - SEI=60：断开后 60 秒内重连，**恢复订阅和离线消息**；60 秒后重连视为新会话。

#### 9. **Message Expiry Interval**
- **描述**：PUBLISH 时设置消息过期时间（单位秒）。
- **判断依据**：
    - 消息在队列中停留超过 expiry 时间，**不再投递给离线客户端**。
    - 在过期前重连，能收到该消息。

#### 10. **Reason Code 与 Reason String**
- **描述**：操作失败时返回标准 Reason Code 和可选 Reason String。
- **判断依据**：
    - 订阅非法 topic（如 `#/#`），SUBACK 返回 **Reason Code = 0x87（Topic Filter Invalid）**。
    - 认证失败，CONNACK 返回 **Code = 0x86（Not Authorized）**。
    - 可通过日志或客户端 API 获取 Reason String（如 "ACL denied"）。

#### 11. **User Properties 透传**
- **描述**：在 CONNECT / PUBLISH / SUBSCRIBE 中携带自定义 key-value。
- **判断依据**：
    - 接收方收到的消息中包含相同的 User Properties（顺序不要求一致）。
    - EMQX 不修改、不丢弃这些属性。

#### 12. **Response Topic 与 Correlation Data（请求-响应模式）**
- **描述**：发布请求消息时携带 response topic 和 correlation data。
- **判断依据**：
    - 请求方在 response topic 上收到响应消息。
    - 响应消息中的 Correlation Data 与请求一致。

#### 13. **Shared Subscription（$share）**
- **描述**：多个客户端订阅 `$share/group1/sensor/+/data`。
- **判断依据**：
    - 向 `sensor/room1/data` 发布消息，**仅 group1 中的一个客户端收到**（负载均衡）。
    - 非共享订阅（如直接订阅 `sensor/+/data`）仍广播给所有订阅者。

#### 14. **Subscription Identifier**
- **描述**：SUBSCRIBE 时携带 Sub ID，消息回带该 ID。
- **判断依据**：
    - 接收方收到 PUBLISH 消息时，**包含对应的 Subscription Identifier**。
    - 多个订阅匹配同一消息时，可收到多个 Sub ID（若支持）。

#### 15. **No Local 订阅选项**
- **描述**：订阅时设置 No Local = true。
- **判断依据**：
    - 客户端向自己订阅的 topic 发布消息，**不会收到自己的消息**。
    - No Local = false（默认）时，会收到。

#### 16. **Retain Handling 控制**
- **描述**：SUBSCRIBE 时设置 Retain Handling（0/1/2）。
- **判断依据**：
    - Retain Handling = 0：总是发送现有 retained 消息。
    - =1：仅当是新订阅（非重复 SUBSCRIBE）时发送。
    - =2：从不发送 retained 消息。
    - 验证首次订阅和重复订阅行为差异。

#### 17. **Maximum Packet Size 协商**
- **描述**：CONNECT 时声明 Maximum Packet Size。
- **判断依据**：
    - 若发布消息超过该值，EMQX **拒绝并断开连接（DISCONNECT with Reason Code 0x95）**。
    - 小于等于限制的消息正常处理。

#### 18. **Receive Maximum 流控**
- **描述**：客户端声明 Receive Maximum（如 10），限制未确认 QoS>0 消息数。
- **判断依据**：
    - EMQX **不会同时发送超过 Receive Maximum 条未确认的 QoS 1/2 消息**。
    - 客户端确认（PUBACK/PUBCOMP）后，才继续发送下一批。

---

### 三、EMQX 特有或增强行为（需验证）

#### 19. **QoS 2 消息持久化与去重**
- **描述**：EMQX 对 QoS 2 消息进行状态跟踪。
- **判断依据**：
    - 网络闪断重连后，QoS 2 消息**不重复、不丢失**。
    - 通过重复 PUBREC/PUBREL 测试幂等性。

#### 20. **离线消息队列长度限制（mqueue）**
- **描述**：EMQX 可配置 per-client 离线消息队列最大长度。
- **判断依据**：
    - 当离线消息超过 limit（如 1000 条），**最早的消息被丢弃**。
    - 重连后收到的是最近的 N 条（按配置）。

---

> ✅ 以上所有测试点均可通过标准 MQTT 客户端（如 paho-mqtt、HiveMQ Client）配合 EMQX Dashboard / 日志 / 规则引擎日志进行验证。