# Cosmic 命令

Cosmic命令用于对Cosmic系统进行巡检操作，通过解析配置文件执行巡检规则。

## 命令格式

```
bbx-cli cosmic [command] [flags]
```

## 命令列表

### inspect - 巡检Cosmic系统

基于配置文件对Cosmic系统进行巡检，解析conf/cosmic目录下的配置文件并执行巡检规则。

**用法：**
```
bbx-cli cosmic inspect [flags]
```

**参数：**
- `-c, --config string`：Cosmic配置文件路径，默认值为 `conf/cosmic/cosmic.yaml`
- `-j, --job string`：要巡检的特定作业名称（可选，不指定则巡检所有作业）
- `-e, --env stringArray`：环境变量覆盖，格式为 `KEY=VALUE`（可选）
- `-o, --output string`：输出结果到文件而不是控制台（可选）
- `-f, --format string`：输出格式，支持 `text|json|yaml|markdown|html|pdf`，默认 `text`

## 功能说明

`cosmic inspect` 命令会：
1. 加载指定的Cosmic配置文件
2. 解析配置文件中的节点信息和作业定义
3. 为每个作业加载对应的巡检规则
4. 按节点维度执行巡检
5. 根据指定格式输出巡检结果

## 示例

```bash
# 基于默认配置文件巡检所有作业
./bbx-cli cosmic inspect

# 指定配置文件进行巡检
./bbx-cli cosmic inspect -c /path/to/custom/cosmic.yaml

# 只巡检特定作业
./bbx-cli cosmic inspect -j admq

# 使用环境变量覆盖配置
./bbx-cli cosmic inspect -e "NODE_ENV=production" -e "LOG_LEVEL=info"

# 输出JSON格式结果到文件
./bbx-cli cosmic inspect -f json -o inspection-results.json

# 输出HTML格式报告
./bbx-cli cosmic inspect -f html -o cosmic-inspection.html

# 输出Markdown格式报告
./bbx-cli cosmic inspect -f markdown -o cosmic-inspection.md

# 输出PDF格式报告（需要安装wkhtmltopdf）
./bbx-cli cosmic inspect -f pdf -o cosmic-inspection.pdf
```

## 配置文件结构

Cosmic配置文件（如 `conf/cosmic/cosmic.yaml`）包含：
- 节点定义：需要巡检的Cosmic节点列表
- 作业定义：每个作业对应要巡检的软件和目标节点
- 规则配置：每个作业对应的巡检规则文件路径

## 输出格式说明

- **text**：人类可读的纯文本格式，带有颜色标记和表格
- **json**：结构化的JSON格式，便于机器处理
- **yaml**：YAML格式，便于阅读和编辑
- **markdown**：Markdown格式，适合生成文档或在GitHub上查看
- **html**：带有样式的HTML格式，包含可折叠的检查结果
- **pdf**：PDF格式，需要安装wkhtmltopdf工具

## 注意事项

- 当使用PDF格式输出时，需要提前安装 `wkhtmltopdf` 工具
- 配置文件路径可以是相对路径或绝对路径
- 环境变量覆盖会应用到所有巡检作业中
- 可以通过 `--job` 参数指定多个作业（多次使用 `-j` 选项）

## 返回码

- 0：所有巡检作业成功完成
- 非0：至少有一个巡检作业失败
