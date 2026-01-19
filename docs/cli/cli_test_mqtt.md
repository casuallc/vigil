# MQTT 集成测试命令

MQTT 测试命令用于测试 MQTT 协议的各种功能，支持 MQTT 3.1.1 和 5.0 版本，以及 EMQX 特定功能。

## 命令格式

```
bbx-cli test mqtt [command]
```

## 命令列表

### all - 运行所有 MQTT 测试

运行所有 MQTT 集成测试。

**用法：**
```
bbx-cli test mqtt all
```

**示例：**
```bash
# 运行所有 MQTT 测试
./bbx-cli test mqtt all
```

## MQTT 3.1.1 测试

### v3 - MQTT 3.1.1 测试命令组

**用法：**
```
bbx-cli test mqtt v3 [command]
```

#### v3 connect - 测试 MQTT 连接

测试 MQTT 客户端连接功能，包括 Clean Session 和 Non-Clean Session。

**用法：**
```
bbx-cli test mqtt v3 connect
```

#### v3 pubsub - 测试 MQTT 发布/订阅

测试 MQTT 发布和订阅功能。

**用法：**
```
bbx-cli test mqtt v3 pubsub
```

#### v3 qos - 测试 MQTT QoS 级别

测试 MQTT QoS 0/1/2 消息传递。

**用法：**
```
bbx-cli test mqtt v3 qos
```

#### v3 retained - 测试 MQTT 保留消息

测试 MQTT 保留消息功能。

**用法：**
```
bbx-cli test mqtt v3 retained
```

#### v3 wildcard - 测试 MQTT 通配符订阅

测试 MQTT 通配符订阅匹配。

**用法：**
```
bbx-cli test mqtt v3 wildcard
```

#### v3 keepalive - 测试 MQTT Keep Alive

测试 MQTT Keep Alive 超时断开连接功能。

**用法：**
```
bbx-cli test mqtt v3 keepalive
```

#### v3 acl - 测试 MQTT ACL 控制

测试 MQTT ACL 控制功能（需要服务器端配置）。

**用法：**
```
bbx-cli test mqtt v3 acl
```

#### v3 tls - 测试 MQTT TLS 连接

测试 MQTT TLS 加密连接（需要服务器端配置）。

**用法：**
```
bbx-cli test mqtt v3 tls
```

## MQTT 5.0 测试

### v5 - MQTT 5.0 测试命令组

**用法：**
```
bbx-cli test mqtt v5 [command]
```

#### v5 session-expiry - 测试 MQTT 5.0 会话过期

测试 MQTT 5.0 会话过期间隔功能。

**用法：**
```
bbx-cli test mqtt v5 session-expiry
```

#### v5 message-expiry - 测试 MQTT 5.0 消息过期

测试 MQTT 5.0 消息过期间隔功能。

**用法：**
```
bbx-cli test mqtt v5 message-expiry
```

#### v5 reason-code - 测试 MQTT 5.0 原因码

测试 MQTT 5.0 原因码和原因字符串功能。

**用法：**
```
bbx-cli test mqtt v5 reason-code
```

#### v5 user-properties - 测试 MQTT 5.0 用户属性

测试 MQTT 5.0 用户属性功能。

**用法：**
```
bbx-cli test mqtt v5 user-properties
```

#### v5 response-topic - 测试 MQTT 5.0 响应主题

测试 MQTT 5.0 响应主题和关联数据功能。

**用法：**
```
bbx-cli test mqtt v5 response-topic
```

#### v5 shared-subscription - 测试 MQTT 5.0 共享订阅

测试 MQTT 5.0 共享订阅功能。

**用法：**
```
bbx-cli test mqtt v5 shared-subscription
```

#### v5 subscription-id - 测试 MQTT 5.0 订阅标识符

测试 MQTT 5.0 订阅标识符功能。

**用法：**
```
bbx-cli test mqtt v5 subscription-id
```

#### v5 no-local - 测试 MQTT 5.0 No Local

测试 MQTT 5.0 No Local 功能。

**用法：**
```
bbx-cli test mqtt v5 no-local
```

#### v5 retain-handling - 测试 MQTT 5.0 Retain Handling

测试 MQTT 5.0 Retain Handling 功能。

**用法：**
```
bbx-cli test mqtt v5 retain-handling
```

#### v5 max-packet-size - 测试 MQTT 5.0 最大数据包大小

测试 MQTT 5.0 最大数据包大小功能。

**用法：**
```
bbx-cli test mqtt v5 max-packet-size
```

#### v5 receive-max - 测试 MQTT 5.0 Receive Maximum

测试 MQTT 5.0 Receive Maximum 功能。

**用法：**
```
bbx-cli test mqtt v5 receive-max
```

## EMQX 特定测试

### emqx - EMQX 特定测试命令组

**用法：**
```
bbx-cli test mqtt emqx [command]
```

#### emqx qos2-persistence - 测试 EMQX QoS 2 消息持久化与去重

测试 EMQX QoS 2 消息持久化和去重功能。

**用法：**
```
bbx-cli test mqtt emqx qos2-persistence
```

#### emqx offline-queue - 测试 EMQX 离线消息队列长度限制

测试 EMQX 离线消息队列长度限制功能。

**用法：**
```
bbx-cli test mqtt emqx offline-queue
```

## 测试结果说明

测试结果将显示每个测试用例的运行状态：
- ✅ 表示测试通过
- ❌ 表示测试失败

最终会显示测试结果汇总，包括总测试数、通过数和失败数。

## 注意事项

1. 运行 MQTT 测试前，请确保本地或远程 MQTT 服务器正在运行，默认连接到 `127.0.0.1:1883`
2. 某些测试需要特定的服务器配置（如 ACL、TLS、MQTT 5.0 支持）
3. 测试过程中会创建临时客户端，测试完成后会自动关闭
4. 建议在测试环境中运行，避免影响生产系统
