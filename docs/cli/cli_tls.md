# TLS 命令文档

## 概述

`tls` 命令组用于管理 TLS 证书，支持生成自签名证书用于 HTTPS。

## 命令结构

```
bbx-cli tls [command]
```

## 命令列表

| 命令 | 描述 |
|------|------|
| `generate` | 生成自签名 TLS 证书 |

## 命令详情

### tls generate

生成自签名 TLS 证书，用于 HTTPS 通信。

**语法：**

```
bbx-cli tls generate --cert <cert_path> --key <key_path> --host <host>
```

**参数：**

| 参数 | 缩写 | 描述 | 必填 | 默认值 |
|------|------|------|------|--------|
| `--cert` | `-c` | 证书文件路径 | 否 | `cert.pem` |
| `--key` | `-k` | 私钥文件路径 | 否 | `key.pem` |
| `--host` | `-H` | 证书的主机名或 IP 地址 | 否 | `localhost` |

**示例：**

```bash
# 使用默认参数生成证书
./bbx-cli tls generate

# 指定证书和密钥路径
./bbx-cli tls generate --cert server.crt --key server.key

# 指定主机名
./bbx-cli tls generate --host example.com

# 完整示例
./bbx-cli tls generate --cert cert.pem --key key.pem --host localhost
```

**输出示例：**

```
TLS certificate generated successfully:
Certificate: cert.pem
Private key: key.pem
```

## 配置 HTTPS

生成证书后，需要在服务器配置文件中添加 HTTPS 配置：

```yaml
https:
  cert_path: "cert.pem"
  key_path: "key.pem"
```

然后启动服务器，它将使用 HTTPS 协议：

```bash
./bbx-server -config path/to/config.yaml
```

客户端可以使用 HTTPS URL 连接到服务器：

```bash
./bbx-cli proc scan -q "MQ" -H https://127.0.0.1:8181
```