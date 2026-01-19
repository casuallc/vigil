# MQTT 集成测试命令

MQTT 测试命令用于测试 MQTT 协议的各种功能，支持 MQTT 3.1.1 和 5.0 版本，以及 EMQX 特定功能。

## 命令格式

```
bbx-cli test mqtt [command]
```

## 全局参数

```
-s, --server string   MQTT server address (default "127.0.0.1")
-p, --port int        MQTT server port (default 1883)
-u, --user string     MQTT username
-P, --password string MQTT password
```

## 命令列表

### all - 运行所有 MQTT 测试

运行所有 MQTT 集成测试。

**用法：**
```
bbx-cli test mqtt all [flags]
```

**示例：**
```bash
# 运行所有 MQTT 测试，连接到指定服务器
./bbx-cli test mqtt all -s 192.168.1.100 -p 1883 -u admin -P password
```

## MQTT 3.1.1 测试

### v3 - MQTT 3.1.1 测试命令组

**用法：**
```
bbx-cli test mqtt v3 [command] [flags]
```

#### v3 connect - 测试 MQTT 连接

测试 MQTT 客户端连接功能，包括 Clean Session 和 Non-Clean Session。

**用法：**
```
bbx-cli test mqtt v3 connect [flags]
```

**示例：**
```bash
# 测试 MQTT 连接
./bbx-cli test mqtt v3 connect

# 连接到指定 MQTT 服务器
./bbx-cli test mqtt v3 connect -s mqtt.example.com -p 1883
```

#### v3 pubsub - 测试 MQTT 发布/订阅

测试 MQTT 发布和订阅功能。

**用法：**
```
bbx-cli test mqtt v3 pubsub [flags]
```

**示例：**
```bash
# 测试 MQTT 发布/订阅功能
./bbx-cli test mqtt v3 pubsub
```

#### v3 qos - 测试 MQTT QoS 级别

测试 MQTT QoS 0/1/2 消息传递。

**用法：**
```
bbx-cli test mqtt v3 qos [flags]
```

**示例：**
```bash
# 测试不同 QoS 级别的消息传递
./bbx-cli test mqtt v3 qos
```

#### v3 retained - 测试 MQTT 保留消息

测试 MQTT 保留消息功能，包括新订阅者接收保留消息、清除保留消息等场景。

**用法：**
```
bbx-cli test mqtt v3 retained [flags]
```

**示例：**
```bash
# 测试 MQTT 保留消息功能
./bbx-cli test mqtt v3 retained
```

#### v3 wildcard - 测试 MQTT 通配符订阅

测试 MQTT 通配符订阅匹配，包括单层通配符 (+) 和多层通配符 (#)。

**用法：**
```
bbx-cli test mqtt v3 wildcard [flags]
```

**示例：**
```bash
# 测试 MQTT 通配符订阅
./bbx-cli test mqtt v3 wildcard
```

#### v3 keepalive - 测试 MQTT Keep Alive

测试 MQTT Keep Alive 功能，验证客户端在指定时间内发送心跳包保持连接。

**用法：**
```
bbx-cli test mqtt v3 keepalive [flags]
```

**示例：**
```bash
# 测试 MQTT Keep Alive 功能
./bbx-cli test mqtt v3 keepalive
```

#### v3 acl - 测试 MQTT ACL 控制

测试 MQTT ACL 控制功能（需要服务器端配置）。

**用法：**
```
bbx-cli test mqtt v3 acl [flags]
```

**示例：**
```bash
# 测试 MQTT ACL 控制
./bbx-cli test mqtt v3 acl
```

#### v3 tls - 测试 MQTT TLS 连接

测试 MQTT TLS 加密连接（需要服务器端配置）。

**用法：**
```
bbx-cli test mqtt v3 tls [flags]
```

**示例：**
```bash
# 测试 MQTT TLS 连接
./bbx-cli test mqtt v3 tls
```

#### v3 lwt - 测试 MQTT 遗嘱消息（LWT）

测试 MQTT 遗嘱消息功能，包括异常断开触发遗嘱、正常断开不触发遗嘱等场景。

**用法：**
```
bbx-cli test mqtt v3 lwt [flags]
```

**示例：**
```bash
# 测试 MQTT 遗嘱消息
./bbx-cli test mqtt v3 lwt
```

#### v3 shared - 测试 MQTT 共享订阅

测试 MQTT 共享订阅功能，验证消息在多个订阅者之间的分发。

**用法：**
```
bbx-cli test mqtt v3 shared [flags]
```

**示例：**
```bash
# 测试 MQTT 共享订阅
./bbx-cli test mqtt v3 shared
```

## MQTT 5.0 测试

### v5 - MQTT 5.0 测试命令组

**用法：**
```
bbx-cli test mqtt v5 [command] [flags]
```

#### v5 session-expiry - 测试 MQTT 5.0 会话过期

测试 MQTT 5.0 会话过期间隔功能。

**用法：**
```
bbx-cli test mqtt v5 session-expiry [flags]
```

**示例：**
```bash
# 测试 MQTT 5.0 会话过期
./bbx-cli test mqtt v5 session-expiry
```

#### v5 message-expiry - 测试 MQTT 5.0 消息过期

测试 MQTT 5.0 消息过期间隔功能。

**用法：**
```
bbx-cli test mqtt v5 message-expiry [flags]
```

**示例：**
```bash
# 测试 MQTT 5.0 消息过期
./bbx-cli test mqtt v5 message-expiry
```

#### v5 reason-code - 测试 MQTT 5.0 原因码

测试 MQTT 5.0 原因码和原因字符串功能。

**用法：**
```
bbx-cli test mqtt v5 reason-code [flags]
```

**示例：**
```bash
# 测试 MQTT 5.0 原因码
./bbx-cli test mqtt v5 reason-code
```

#### v5 user-properties - 测试 MQTT 5.0 用户属性

测试 MQTT 5.0 用户属性功能。

**用法：**
```
bbx-cli test mqtt v5 user-properties [flags]
```

**示例：**
```bash
# 测试 MQTT 5.0 用户属性
./bbx-cli test mqtt v5 user-properties
```

#### v5 response-topic - 测试 MQTT 5.0 响应主题

测试 MQTT 5.0 响应主题和关联数据功能。

**用法：**
```
bbx-cli test mqtt v5 response-topic [flags]
```

**示例：**
```bash
# 测试 MQTT 5.0 响应主题
./bbx-cli test mqtt v5 response-topic
```

#### v5 shared-subscription - 测试 MQTT 5.0 共享订阅

测试 MQTT 5.0 共享订阅功能。

**用法：**
```
bbx-cli test mqtt v5 shared-subscription [flags]
```

**示例：**
```bash
# 测试 MQTT 5.0 共享订阅
./bbx-cli test mqtt v5 shared-subscription
```

#### v5 subscription-id - 测试 MQTT 5.0 订阅标识符

测试 MQTT 5.0 订阅标识符功能。

**用法：**
```
bbx-cli test mqtt v5 subscription-id [flags]
```

**示例：**
```bash
# 测试 MQTT 5.0 订阅标识符
./bbx-cli test mqtt v5 subscription-id
```

#### v5 no-local - 测试 MQTT 5.0 No Local

测试 MQTT 5.0 No Local 功能。

**用法：**
```
bbx-cli test mqtt v5 no-local [flags]
```

**示例：**
```bash
# 测试 MQTT 5.0 No Local
./bbx-cli test mqtt v5 no-local
```

#### v5 retain-handling - 测试 MQTT 5.0 Retain Handling

测试 MQTT 5.0 Retain Handling 功能。

**用法：**
```
bbx-cli test mqtt v5 retain-handling [flags]
```

**示例：**
```bash
# 测试 MQTT 5.0 Retain Handling
./bbx-cli test mqtt v5 retain-handling
```

#### v5 max-packet-size - 测试 MQTT 5.0 最大数据包大小

测试 MQTT 5.0 最大数据包大小功能。

**用法：**
```
bbx-cli test mqtt v5 max-packet-size [flags]
```

**示例：**
```bash
# 测试 MQTT 5.0 最大数据包大小
./bbx-cli test mqtt v5 max-packet-size
```

#### v5 receive-max - 测试 MQTT 5.0 Receive Maximum

测试 MQTT 5.0 Receive Maximum 功能。

**用法：**
```
bbx-cli test mqtt v5 receive-max [flags]
```

**示例：**
```bash
# 测试 MQTT 5.0 Receive Maximum
./bbx-cli test mqtt v5 receive-max
```

## EMQX 特定测试

### emqx - EMQX 特定测试命令组

**用法：**
```
bbx-cli test mqtt emqx [command] [flags]
```

#### emqx qos2-persistence - 测试 EMQX QoS 2 消息持久化与去重

测试 EMQX QoS 2 消息持久化和去重功能。

**用法：**
```
bbx-cli test mqtt emqx qos2-persistence [flags]
```

**示例：**
```bash
# 测试 EMQX QoS 2 消息持久化与去重
./bbx-cli test mqtt emqx qos2-persistence
```

#### emqx offline-queue - 测试 EMQX 离线消息队列长度限制

测试 EMQX 离线消息队列长度限制功能。

**用法：**
```
bbx-cli test mqtt emqx offline-queue [flags]
```

**示例：**
```bash
# 测试 EMQX 离线消息队列长度限制
./bbx-cli test mqtt emqx offline-queue
```

## 测试结果说明

测试结果将显示每个测试用例的运行状态：
- ✅ 表示测试通过
- ❌ 表示测试失败

最终会显示测试结果汇总，包括：
- 总测试数
- 通过测试数
- 失败测试数

## 注意事项

1. 运行 MQTT 测试前，请确保本地或远程 MQTT 服务器正在运行，默认连接到 `127.0.0.1:1883`
2. 某些测试需要特定的服务器配置：
   - ACL 测试：需要 MQTT 服务器配置了 ACL 规则
   - TLS 测试：需要 MQTT 服务器配置了 TLS 证书
   - MQTT 5.0 测试：需要服务器支持 MQTT 5.0 协议
3. 测试过程中会创建临时客户端，测试完成后会自动关闭
4. 建议在测试环境中运行，避免影响生产系统
5. 可以使用全局参数 `-s`, `-p`, `-u`, `-P` 配置连接信息

## 测试用例说明

### 保留消息测试用例

- **RET-01**: 新订阅者收到保留消息
- **RET-02**: 发布空payload清除保留消息
- **RET-03**: 非retain消息不影响保留消息
- **RET-04**: 新retain消息替换旧保留消息
- **RET-08**: 单层通配符接收保留消息
- **RET-09**: 多层通配符接收保留消息

### 通配符订阅测试用例

#### 单层通配符 '+' 测试
- **T1-01**: Basic match
- **T1-02**: Extra level no match
- **T1-03**: Missing level no match
- **T1-04**: Start with +
- **T1-05**: End with +
- **T1-06**: Empty level no match
- **T1-07**: Multiple +
- **T1-08**: Match numbers
- **T1-09**: Case sensitivity

#### 多层通配符 '#' 测试
- **T2-01**: Match self
- **T2-02**: Match one level
- **T2-03**: Match deep path
- **T2-04**: Prefix mismatch
- **T2-05**: Exact match also works
- **T2-06**: Global subscribe
- **T2-07**: # with / boundary

#### 系统主题 $SYS 隔离验证
- **T4-01**: # does not match $SYS
- **T4-02**: Explicit $SYS subscribe
- **T4-03**: +/+ does not match $SYS

#### 边界与特殊字符
- **T6-01**: Levels with special characters
- **T6-03**: Single character levels

### 遗嘱消息测试用例

- **LWT-01**: 客户端异常断开，触发遗嘱
- **LWT-02**: 客户端正常DISCONNECT，不触发遗嘱
- **LWT-05**: 遗嘱retain=true，新订阅者可收到
- **LWT-06**: 遗嘱retain=false，新订阅者收不到
- **LWT-11**: Clean Session=true，异常断开仍触发遗嘱
- **LWT-12**: Clean Session=false，异常断开仍触发遗嘱
- **LWT-13**: 订阅者使用通配符接收遗嘱
- **LWT-14**: 遗嘱主题含特殊字符（合法）
