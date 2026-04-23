# Hosts 文件管理 API

## 接口列表

| 接口路径 | 请求方法 | 功能描述 |
|---------|----------|----------|
| /api/hosts | POST | 更新 /etc/hosts 文件条目 |

---

## POST /api/hosts

**功能描述**：批量更新系统 hosts 文件条目。按 IP 判断是否存在：如果 IP 已存在则替换整行，不存在则追加到文件末尾。

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| ip | string | 是 | IP 地址 |
| hostname | string | 是 | 主机名 |

**请求示例**：
```json
[
  {"ip": "192.168.1.10", "hostname": "myhost"},
  {"ip": "192.168.1.20", "hostname": "anotherhost"}
]
```

**响应格式**（成功）：
```json
{
  "message": "Hosts file updated successfully"
}
```

**错误响应**：
```json
{
  "error": "entry 0: ip is required"
}
```

**字段说明**：

| 字段 | 类型 | 说明 |
|------|------|------|
| message | string | 操作成功提示 |

**注意事项**：

- 运行服务器需要具备写入 hosts 文件的权限（Linux/macOS 为 `/etc/hosts`，Windows 为 `C:\Windows\System32\drivers\etc\hosts`）
- 原有注释行和空行会被保留
- 重复的 IP 会被去重（后面的覆盖前面的）
