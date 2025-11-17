## 巡检命令

### 命令格式

```bash
vigil cosmic inspect [flags]
```

### 命令说明

对cosmic系统进行巡检，根据配置文件中的规则检查各个节点的状态。

### 参数说明

| 参数名 | 简写 | 是否必须 | 参数说明 |
| ------ | ---- | -------- | -------- |
| --config | -c | 否 | cosmic配置文件路径，默认为conf/cosmic/cosmic.yaml |
| --job | -j | 否 | 指定要巡检的作业名称，如果不指定则检查所有作业 |
| --env | -e | 否 | 环境变量，可以多次使用，格式为KEY=VALUE |
| --format | -f | 否 | 输出格式 (text, yaml, json)，默认为text |
| --output | -o | 否 | 输出结果到指定文件，如果不指定则输出到控制台 |

### 使用示例

```bash
# 使用默认配置检查所有作业
vigil cosmic inspect

# 检查指定作业
vigil cosmic inspect --job admq

# 指定配置文件并输出JSON格式
vigil cosmic inspect --config ./my-cosmic-config.yaml --format json

# 设置环境变量并输出到文件
vigil cosmic inspect --env USER=admin --env PASSWORD=secret --output report.txt

# 检查指定作业并输出YAML格式到文件
vigil cosmic inspect --job amdc --format yaml --output amdc-report.yaml
```

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