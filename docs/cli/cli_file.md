# File 命令文档

## 概述

`file` 命令组用于执行文件操作，包括上传、下载、列出、删除、复制和移动文件。

## 命令结构

```
bbx-cli file [command]
```

## 命令列表

| 命令 | 描述 |
|------|------|
| `upload` | 上传本地文件 |
| `download` | 下载文件到本地 |
| `list` | 列出目录中的文件 |
| `delete` | 删除文件 |
| `copy` | 复制文件 |
| `move` | 移动文件 |

## 命令详情

### file upload

上传本地文件到服务器。

**语法：**

```
bbx-cli file upload --source <source_path> --target <target_path>
```

**参数：**

| 参数 | 缩写 | 描述 | 必填 | 默认值 |
|------|------|------|------|--------|
| `--source` | `-s` | 源文件路径 | 是 | - |
| `--target` | `-t` | 目标文件路径 | 是 | - |

**示例：**

```bash
# 上传本地文件到服务器
./bbx-cli file upload --source local.txt --target remote.txt
```

### file download

从服务器下载文件到本地。

**语法：**

```
bbx-cli file download --source <source_path> --target <target_path>
```

**参数：**

| 参数 | 缩写 | 描述 | 必填 | 默认值 |
|------|------|------|------|--------|
| `--source` | `-s` | 源文件路径 | 是 | - |
| `--target` | `-t` | 目标文件路径 | 是 | - |

**示例：**

```bash
# 从服务器下载文件到本地
./bbx-cli file download --source remote.txt --target local.txt
```

### file list

列出服务器目录中的文件。

**语法：**

```
bbx-cli file list --path <path> --max-depth <depth>
```

**参数：**

| 参数 | 缩写 | 描述 | 必填 | 默认值 |
|------|------|------|------|--------|
| `--path` | `-p` | 目录路径 | 否 | `/` |
| `--max-depth` | `-d` | 递归列出的最大深度（0 表示不递归） | 否 | 0 |

**示例：**

```bash
# 列出根目录中的文件
./bbx-cli file list

# 列出指定目录中的文件，递归深度为 2
./bbx-cli file list --path /home --max-depth 2
```

### file delete

删除服务器上的文件。

**语法：**

```
bbx-cli file delete --path <path>
```

**参数：**

| 参数 | 缩写 | 描述 | 必填 | 默认值 |
|------|------|------|------|--------|
| `--path` | `-p` | 文件路径 | 是 | - |

**示例：**

```bash
# 删除服务器上的文件
./bbx-cli file delete --path remote.txt
```

### file copy

在服务器上复制文件。

**语法：**

```
bbx-cli file copy --source <source_path> --target <target_path>
```

**参数：**

| 参数 | 缩写 | 描述 | 必填 | 默认值 |
|------|------|------|------|--------|
| `--source` | `-s` | 源文件路径 | 是 | - |
| `--target` | `-t` | 目标文件路径 | 是 | - |

**示例：**

```bash
# 在服务器上复制文件
./bbx-cli file copy --source source.txt --target target.txt
```

### file move

在服务器上移动文件。

**语法：**

```
bbx-cli file move --source <source_path> --target <target_path>
```

**参数：**

| 参数 | 缩写 | 描述 | 必填 | 默认值 |
|------|------|------|------|--------|
| `--source` | `-s` | 源文件路径 | 是 | - |
| `--target` | `-t` | 目标文件路径 | 是 | - |

**示例：**

```bash
# 在服务器上移动文件
./bbx-cli file move --source old.txt --target new.txt
```