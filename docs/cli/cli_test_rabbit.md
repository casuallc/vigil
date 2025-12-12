# RabbitMQ 集成测试命令

RabbitMQ 测试命令用于测试 RabbitMQ 协议的各种功能，包括消息发布、交换器路由、队列绑定、消息消费、死信队列、消息 TTL、消费者并发和发布者确认等功能。

## 命令格式

```
bbx-cli test rabbit [command]
```

## 全局参数

```
-s, --server string   RabbitMQ server address (default "127.0.0.1")
-p, --port int        RabbitMQ server port (default 5672)
-V, --vhost string    RabbitMQ vhost (default "/")
-u, --user string     RabbitMQ username (default "guest")
-P, --password string RabbitMQ password (default "guest")
```

## 命令列表

### all - 运行所有 RabbitMQ 测试

运行所有 RabbitMQ 集成测试。

**用法：**
```
bbx-cli test rabbit all [flags]
```

**示例：**
```bash
# 运行所有 RabbitMQ 测试，连接到默认服务器
./bbx-cli test rabbit all

# 运行所有 RabbitMQ 测试，连接到指定服务器
./bbx-cli test rabbit all -s rabbit.example.com -p 5672 -u admin -P password -V /test
```

### publish - 测试消息发布可靠性

测试 RabbitMQ 消息发布可靠性，包括基本发布、发布到默认 Exchange、发布到不存在的 Exchange、持久化消息、带自定义 headers 等场景。

**用法：**
```
bbx-cli test rabbit publish [flags]
```

**示例：**
```bash
# 测试消息发布可靠性
./bbx-cli test rabbit publish

# 连接到指定 RabbitMQ 服务器进行测试
./bbx-cli test rabbit publish -s 192.168.1.100 -p 5672
```

### routing - 测试交换器路由规则

测试不同类型交换器（direct、topic、fanout）的路由规则。

**用法：**
```
bbx-cli test rabbit routing [flags]
```

**示例：**
```bash
# 测试交换器路由规则
./bbx-cli test rabbit routing
```

### binding - 测试队列绑定

测试 RabbitMQ 队列绑定和解绑功能。

**用法：**
```
bbx-cli test rabbit binding [flags]
```

**示例：**
```bash
# 测试队列绑定
./bbx-cli test rabbit binding
```

### consume - 测试消息消费和确认

测试 RabbitMQ 消息消费和确认功能，包括自动确认和手动确认。

**用法：**
```
bbx-cli test rabbit consume [flags]
```

**示例：**
```bash
# 测试消息消费和确认
./bbx-cli test rabbit consume
```

### dlq - 测试死信队列

测试 RabbitMQ 死信队列机制，包括消息被拒绝进入死信队列等场景。

**用法：**
```
bbx-cli test rabbit dlq [flags]
```

**示例：**
```bash
# 测试死信队列
./bbx-cli test rabbit dlq
```

### ttl - 测试消息 TTL

测试 RabbitMQ 消息 TTL（Time-to-Live）功能，验证消息在指定时间后过期。

**用法：**
```
bbx-cli test rabbit ttl [flags]
```

**示例：**
```bash
# 测试消息 TTL
./bbx-cli test rabbit ttl
```

### concurrency - 测试消费者并发

测试 RabbitMQ 消费者并发和公平调度功能。

**用法：**
```
bbx-cli test rabbit concurrency [flags]
```

**示例：**
```bash
# 测试消费者并发
./bbx-cli test rabbit concurrency
```

### confirms - 测试发布者确认

测试 RabbitMQ 发布者确认功能，确保消息被服务器正确接收。

**用法：**
```
bbx-cli test rabbit confirms [flags]
```

**示例：**
```bash
# 测试发布者确认
./bbx-cli test rabbit confirms
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

1. 运行 RabbitMQ 测试前，请确保本地或远程 RabbitMQ 服务器正在运行，默认连接到 `127.0.0.1:5672`
2. 测试过程中会创建临时 Exchange、Queue 和 Binding，测试完成后会自动清理
3. 建议在测试环境中运行，避免影响生产系统
4. 可以使用全局参数 `-s`, `-p`, `-V`, `-u`, `-P` 配置连接信息
5. 默认使用 guest/guest 用户名密码，部分 RabbitMQ 服务器可能禁用了默认用户

## 测试用例说明

### 消息发布测试用例

- **RB-PUB-01**: 基本消息发布
- **RB-PUB-02**: 发布到默认 Exchange
- **RB-PUB-03**: 发布到不存在的 Exchange
- **RB-PUB-04**: 持久化消息发布
- **RB-PUB-05**: 非持久化消息发布
- **RB-PUB-06**: 带自定义 headers
- **RB-PUB-07**: 带 content-type

### 交换器路由测试用例

- **RB-ROUT-01**: Direct Exchange 精确匹配
- **RB-ROUT-02**: 不匹配路由
- **RB-ROUT-03**: 多队列匹配相同键
- **RB-ROUT-04**: Topic Exchange 精确主题匹配
- **RB-ROUT-05**: 单层通配符匹配
- **RB-ROUT-06**: 多层通配符匹配
- **RB-ROUT-07**: 多层通配符不匹配
- **RB-ROUT-09**: 广播到所有绑定队列
- **RB-ROUT-10**: 忽略路由键

### 队列绑定测试用例

- **RB-BIND-01**: 队列绑定到交换器

### 消息消费测试用例

- **RB-CONS-01**: 自动确认消费
- **RB-CONS-02**: 手动确认消费

### 死信队列测试用例

- **RB-DLQ-01**: 消息被拒绝进入DLQ

### 消息 TTL 测试用例

- **RB-TTL-01**: 消息TTL过期

### 消费者并发测试用例

- **RB-CONCUR-01**: 多消费者并发消费

### 发布者确认测试用例

- **RB-CONF-01**: 发布者确认测试
