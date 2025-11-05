## 可视化命令

### 命令格式

```bash
vigil cosmic visual [flags]
```

### 命令说明

以更直观的方式展示巡检报告，支持HTML和终端彩色输出。

### 参数说明

| 参数名 | 简写 | 是否必须 | 参数说明 |
| ------ | ---- | -------- | -------- |
| --report | -r | 是 | 巡检报告文件路径 |
| --format | -f | 否 | 输出格式 (terminal, html)，默认为terminal |

### 使用示例

```bash
# 以终端彩色方式展示巡检报告
vigil cosmic visual --report ./reports/cosmic/report_20230601_120000.json

# 生成HTML格式的可视化报告
vigil cosmic visual --report ./reports/cosmic/report_20230601_120000.json --format html
```