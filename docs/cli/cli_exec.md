# 命令执行命令

exec命令用于在服务器上执行命令或脚本。

## 命令格式

```
bbx-cli exec [command/script] [flags]
```

## 命令参数

- `command/script`：要执行的命令或脚本路径（必填）
- `-f, --file`：将参数视为脚本文件路径
- `-e, --env stringArray`：要设置的环境变量（格式：KEY=VALUE）
- `-r, --result string`：将结果输出到文件而不是控制台

## 示例

```bash
# 执行简单命令
./bbx-cli exec "ls -la" -H http://127.0.0.1:8181

# 执行脚本文件
./bbx-cli exec "/path/to/script.sh" -f -H http://127.0.0.1:8181

# 执行带环境变量的命令
./bbx-cli exec "echo $MY_VAR" -e MY_VAR=value -H http://127.0.0.1:8181

# 将结果输出到文件
./bbx-cli exec "cat /etc/passwd" -r result.txt -H http://127.0.0.1:8181
```
