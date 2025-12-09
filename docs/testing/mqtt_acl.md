## 🧪 MQTT ACL 控制集成测试用例

> **测试前提**：
> - 使用 MQTT 3.1.1 或 5.0 协议
> - Broker：EMQX（已配置 ACL 规则）
> - 每个测试使用独立客户端
> - 已配置以下 ACL 规则：
>   - 允许用户 `allowed_user` 发布/订阅 `allowed/topic/#`
>   - 拒绝用户 `denied_user` 访问任何资源
>   - 只允许用户 `publish_only` 发布 `publish/only/#`
>   - 只允许用户 `subscribe_only` 订阅 `subscribe/only/#`

---

### ✅ 分类一：基本 ACL 功能验证

| ID | 描述 | 用户名 | 操作 | 主题 | 预期结果 | 说明 |
|----|------|--------|------|------|--------|------|
| ACL-01 | 允许的用户和主题 | `allowed_user` | 发布 | `allowed/topic/message` | ✅ 发布成功 | 匹配允许规则 |
| ACL-02 | 允许的用户和主题 | `allowed_user` | 订阅 | `allowed/topic/#` | ✅ 订阅成功 | 匹配允许规则 |
| ACL-03 | 拒绝的用户 | `denied_user` | 发布 | `any/topic` | ❌ 发布失败（CONNACK 或 PUBLISH 被拒绝） | 用户被完全拒绝 |
| ACL-04 | 拒绝的用户 | `denied_user` | 订阅 | `any/topic` | ❌ 订阅失败（SUBACK 失败） | 用户被完全拒绝 |

---

### ✅ 分类二：发布与订阅权限分离

| ID | 描述 | 用户名 | 操作 | 主题 | 预期结果 |
|----|------|--------|------|------|--------|
| ACL-05 | 只允许发布 | `publish_only` | 发布 | `publish/only/message` | ✅ 发布成功 |
| ACL-06 | 只允许发布尝试订阅 | `publish_only` | 订阅 | `publish/only/#` | ❌ 订阅失败 |
| ACL-07 | 只允许订阅 | `subscribe_only` | 订阅 | `subscribe/only/#` | ✅ 订阅成功 |
| ACL-08 | 只允许订阅尝试发布 | `subscribe_only` | 发布 | `subscribe/only/message` | ❌ 发布失败 |

---

### ✅ 分类三：ACL 与主题匹配

| ID | 描述 | 用户名 | 操作 | 主题 | 预期结果 |
|----|------|--------|------|------|--------|
| ACL-09 | 精确主题匹配 | `allowed_user` | 发布 | `allowed/topic/exact` | ✅ 发布成功 |
| ACL-10 | 通配符主题匹配 | `allowed_user` | 订阅 | `allowed/topic/+` | ✅ 订阅成功 |
| ACL-11 | 超出权限范围的主题 | `allowed_user` | 发布 | `restricted/topic` | ❌ 发布失败 |
| ACL-12 | 子主题访问 | `allowed_user` | 发布 | `allowed/topic/sub/level` | ✅ 发布成功 | 继承父主题权限 |

---

### ✅ 分类四：ACL 与连接状态

| ID | 描述 | 用户名 | 密码 | 预期结果 | 说明 |
|----|------|--------|------|--------|------|
| ACL-13 | 正确用户名密码 | `allowed_user` | `correct_password` | ✅ 连接成功 | 身份验证通过 |
| ACL-14 | 错误密码 | `allowed_user` | `wrong_password` | ❌ 连接失败 | 身份验证失败 |
| ACL-15 | 不存在的用户 | `non_existent_user` | `any_password` | ❌ 连接失败 | 用户不存在 |

---

### ✅ 分类五：MQTT 5.0 ACL 扩展

| ID | 描述 | 用户名 | 操作 | 主题 | 预期结果 |
|----|------|--------|------|------|--------|
| ACL-16 | MQTT 5.0 ACL 拒绝原因码 | `denied_user` | 订阅 | `any/topic` | ❌ SUBACK 失败，Reason Code=0x87 (Not Authorized) | MQTT 5.0 提供拒绝原因 |
| ACL-17 | MQTT 5.0 连接拒绝原因码 | `denied_user` | 连接 | — | ❌ CONNACK 失败，Reason Code=0x86 (Bad Username or Password) |

---

## ✅ 总结：覆盖维度

| 维度 | 是否覆盖 |
|------|--------|
| 基本 ACL 允许/拒绝 | ✅ |
| 发布与订阅权限分离 | ✅ |
| 主题匹配规则 | ✅ |
| 身份验证 | ✅ |
| MQTT 5.0 扩展 | ✅ |
| 错误处理 | ✅ |

---