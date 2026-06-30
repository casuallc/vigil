# Transfer 命令文档

## 概述

`transfer` 命令组用于操作文件传输 Agent：浏览 Agent 所在主机的文件系统，以及创建、管理分块文件传输任务。命令通过 REST API 与 `bbx-server` 通信。

> 这些命令访问 Agent 自有认证的 `/api/transfer/*`、`/api/fs/*` 接口。CLI 会从 `conf/config.yaml` 的 `filetransfer.auth_user` / `filetransfer.auth_pass` 读取凭据（默认 `admq` / `admq-file-transfer`），与全局 `auth` 凭据相互独立。需确保 `filetransfer.enabled: true`。

## 命令结构

```
bbx-cli transfer [command]
```

## 命令列表

| 命令 | 描述 |
|------|------|
| `fs list` | 列出 Agent 上的目录内容 |
| `fs stat` | 查看文件/目录统计信息 |
| `task create` | 从 TaskConfig JSON 文件创建任务 |
| `task list` | 列出所有任务 |
| `task get` | 获取任务配置 |
| `task delete` | 删除任务 |
| `task start` | 启动任务 |
| `task pause` | 暂停任务 |
| `task resume` | 恢复任务 |
| `task cancel` | 取消任务 |
| `task status` | 查看任务聚合状态 |
| `task progress` | 查看每文件进度 |

## 命令详情

### transfer fs list

列出 Agent 上指定目录的内容（受路径监狱约束）。

**语法：**

```
bbx-cli transfer fs list --path <dir>
```

**参数：**

| 参数 | 描述 | 必填 |
|------|------|------|
| `--path` | 要列出的目录路径 | 是 |

**示例：**

```bash
./bbx-cli transfer fs list --path /data/logs
```

**输出示例：**

```
TYPE  SIZE  NAME
dir   0     archive
file  1024  app.log
```

---

### transfer fs stat

查看文件或目录的统计信息（目录会递归汇总文件数与总字节数），以 JSON 输出。

**语法：**

```
bbx-cli transfer fs stat --path <path>
```

**参数：**

| 参数 | 描述 | 必填 |
|------|------|------|
| `--path` | 要查看的文件或目录路径 | 是 |

**示例：**

```bash
./bbx-cli transfer fs stat --path /data/logs
```

**输出示例：**

```json
{
  "isDir": true,
  "size": 0,
  "fileCount": 12,
  "totalSize": 1048576
}
```

---

### transfer task create

从一个完整的 `TaskConfig` JSON 文件创建任务。

**语法：**

```
bbx-cli transfer task create --file <task.json>
```

**参数：**

| 参数 | 缩写 | 描述 | 必填 |
|------|------|------|------|
| `--file` | `-f` | TaskConfig JSON 文件路径 | 是 |

**task.json 示例**（SEND DIRECT，向对端 Agent 推送一个目录）：

```json
{
  "taskId": 1001,
  "role": "SEND",
  "relayType": "DIRECT",
  "sourcePaths": ["/data/release"],
  "chunkSize": 1048576,
  "overwritePolicy": "OVERWRITE",
  "targets": [
    {
      "host": "10.0.0.2",
      "port": 57575,
      "authUser": "admq",
      "authPass": "admq-file-transfer",
      "agentTaskId": 2001
    }
  ]
}
```

**task.json 示例**（RECV DIRECT，落地到本机目录）：

```json
{
  "taskId": 2001,
  "role": "RECV",
  "relayType": "DIRECT",
  "targetDir": "/data/incoming",
  "overwritePolicy": "RENAME"
}
```

**示例：**

```bash
./bbx-cli transfer task create -f task.json
```

**输出示例：**

```
Task 1001 created
```

---

### transfer task list

列出所有任务。

**语法：**

```
bbx-cli transfer task list
```

**输出示例：**

```
ID    ROLE  RELAY   TARGET_DIR
1001  SEND  DIRECT
2001  RECV  DIRECT  /data/incoming
```

---

### transfer task get / status / progress

按 ID 查询任务，结果以 JSON 输出。

**语法：**

```
bbx-cli transfer task get      --id <id>
bbx-cli transfer task status   --id <id>
bbx-cli transfer task progress --id <id>
```

**参数：**

| 参数 | 描述 | 必填 |
|------|------|------|
| `--id` | 任务 ID | 是 |

**`task status` 输出示例：**

```json
{
  "taskId": 1001,
  "state": "RUNNING",
  "progress": 42,
  "totalBytes": 10485760,
  "transferredBytes": 4404019,
  "totalFiles": 3,
  "completedFiles": 1,
  "files": [
    { "relPath": "a.bin", "receivedBytes": 1048576, "totalBytes": 1048576, "completed": true }
  ],
  "errorMsg": ""
}
```

---

### transfer task start / pause / resume / cancel

驱动任务生命周期。

**语法：**

```
bbx-cli transfer task start  --id <id>
bbx-cli transfer task pause  --id <id>
bbx-cli transfer task resume --id <id>
bbx-cli transfer task cancel --id <id>
```

**参数：**

| 参数 | 描述 | 必填 |
|------|------|------|
| `--id` | 任务 ID | 是 |

**状态约束：**

| 操作 | 允许的前置状态 |
|------|----------------|
| `start` | IDLE、PAUSED |
| `pause` | RUNNING |
| `resume` | PAUSED |
| `cancel` | 任意 |

状态非法时返回错误（例如对 RUNNING 任务执行 `start`）。

**示例：**

```bash
./bbx-cli transfer task start --id 1001
./bbx-cli transfer task pause --id 1001
./bbx-cli transfer task resume --id 1001
```

**输出示例：**

```
Task 1001 started
```

---

### transfer task delete

取消任务（若在运行）并删除其本地持久化数据。

**语法：**

```
bbx-cli transfer task delete --id <id>
```

**示例：**

```bash
./bbx-cli transfer task delete --id 1001
```

**输出示例：**

```
Task 1001 deleted
```

## 完整示例：两端 DIRECT 传输

```bash
# 接收端（10.0.0.2）：创建并启动 RECV 任务
./bbx-cli transfer task create -f recv.json -H http://10.0.0.2:57575
./bbx-cli transfer task start --id 2001 -H http://10.0.0.2:57575

# 发送端（10.0.0.1）：创建并启动 SEND 任务（targets 指向接收端）
./bbx-cli transfer task create -f send.json -H http://10.0.0.1:57575
./bbx-cli transfer task start --id 1001 -H http://10.0.0.1:57575

# 查看发送进度
./bbx-cli transfer task status --id 1001 -H http://10.0.0.1:57575
```
