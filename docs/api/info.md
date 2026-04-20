# 服务器信息 API

## 接口列表

| 接口路径 | 请求方法 | 功能描述 |
|---------|----------|----------|
| /api/info | GET | 获取服务器基本信息 |

---

## GET /api/info

**功能描述**：获取服务器主机名、网卡信息、程序版本、CPU 架构、操作系统等信息

**请求参数**：无

**响应格式**：
```json
{
  "hostname": "my-server",
  "interfaces": [
    {
      "mac": "00:16:3e:12:34:56",
      "network": "192.168.1.2/24",
      "ip": "192.168.1.2"
    }
  ],
  "version": {
    "Version": "1.0",
    "BuildTime": "2025-04-20T10:00:00Z",
    "GitCommit": "abc1234",
    "GitBranch": "master",
    "GoVersion": "go1.24.0",
    "OS": "linux",
    "Arch": "amd64"
  },
  "arch": "amd64",
  "os": "linux"
}
```

**字段说明**：

| 字段 | 类型 | 说明 |
|------|------|------|
| hostname | string | 服务器主机名 |
| interfaces | array | 在线非回环网卡列表 |
| interfaces[].mac | string | 网卡 MAC 地址 |
| interfaces[].network | string | 网络地址（CIDR 格式） |
| interfaces[].ip | string | IP 地址 |
| version | object | 程序版本信息 |
| arch | string | CPU 架构（如 amd64、arm64） |
| os | string | 操作系统（如 linux、windows、darwin） |
