# 龙芯 LoongArch 旧世界（麒麟 V10 / UOS V20 / Loongnix 20）适配指南

> 适用场景：在**麒麟 V10、UOS V20、Loongnix 20** 等龙芯「旧世界」系统上部署/开发
> Go 程序时遇到**启动即段错误**、**`masking profiler signal: invalid argument` panic**
> 等问题。本文档记录根因、本项目（vigil/bbx）的完整解决方案，以及排查其它程序
> 或新增依赖时同类问题的通用套路。

## 1. 背景：龙芯的「新旧世界」

龙芯 LoongArch 存在两套互不兼容的软件 ABI：

| | 旧世界 (ABI1.0) | 新世界 (ABI2.0) |
|---|---|---|
| 代表系统 | 麒麟 V10、UOS V20、Loongnix 20 | 麒麟 V11、UOS V25、Loongnix 25、openEuler、Arch 等 |
| 内核 | 4.19 / 5.4 / 5.10 | ≥ 5.19，常见 ≥ 6.1 |
| 动态加载器 | `/lib64/ld.so.1` | `/lib64/ld-linux-loongarch-lp64d.so.1` |
| Go 工具链 | **龙芯魔改工具链**（abi1.0） | **上游 Go**（go ≥ 1.19） |
| ELF e_flags | `0x3, DOUBLE-FLOAT, OBJ-v0` | `0x43, DOUBLE-FLOAT, OBJ-v1` |

两个世界的程序**不能互相运行**，也没有新世界→旧世界的兼容层
（AOSC 的 libLoL 只能让旧世界程序跑在新世界系统上，反方向无解）。

## 2. 快速判断

### 2.1 目标系统属于哪个世界

```bash
# 方法 1：看内核版本（4.19/5.4/5.10 → 旧世界）
uname -r

# 方法 2：看动态加载器
ls /lib64/ld.so.1 /lib64/ld-linux-loongarch-lp64d.so.1 2>/dev/null
# 只有前者 → 旧世界；有后者 → 新世界

# 方法 3：看系统自带二进制的 ELF 标志（offset 48 即 e_flags）
hexdump -s 48 -C /usr/bin/sh | head -n 1
# 第 6 列为 03 → 旧世界；43 → 新世界
```

### 2.2 一个二进制属于哪个世界

```bash
readelf -h ./bbx-server | grep Flags
# Flags: 0x3, DOUBLE-FLOAT, OBJ-v0  → 旧世界 (ABI1.0)
# Flags: 0x43, DOUBLE-FLOAT, OBJ-v1 → 新世界 (ABI2.0)
```

注意：`file` 命令对静态链接的 Go 程序**看不出**新旧世界（都显示
`statically linked`），必须看 e_flags。

## 3. 典型症状与根因

### 3.1 启动即段错误（最常见）

```
$ ./bbx-server
段错误（核心已转储）
```

**根因**：用上游 Go 交叉编译的 loong64 二进制是**新世界**程序。Go runtime
初始化时必须调用 `rt_sigprocmask`，其使用的 sigset 大小（8 字节）与旧世界
内核要求（16 字节，NSIG=128）不一致，系统调用返回失败。Go runtime 认为
「必然成功的系统调用居然失败，内核服务已不可靠」，**故意访问非法地址自杀**。

任何上游 Go 编译的 loong64 程序（包括 hello world）在旧世界都是这个表现，
与业务代码无关。

### 3.2 `panic: masking profiler signal: invalid argument`

```
panic: masking profiler signal: invalid argument
github.com/cilium/ebpf/internal/sys.maskProfilerSignal()
    .../cilium/ebpf@*/internal/sys/signals.go:30
```

**根因**：即使用了龙芯 abi1.0 工具链（runtime 已修补，程序能启动），
**第三方依赖**里的 `golang.org/x/sys/unix` 仍是上游版本。`x/sys` 对 loong64
硬编码 `_C__NSIG = 0x41`，其 `PthreadSigmask`（`syscall_linux.go`）传给
`rt_sigprocmask` 的 sigsetsize = `0x41/8` = **8**，旧世界内核要求 **16**，
返回 `EINVAL`，cilium/ebpf 直接 panic。

同类隐患（都因 sigsetsize=8 在旧世界会失败）：

- `unix.PthreadSigmask` → `rt_sigprocmask`
- `unix.Signalfd` → `signalfd4`
- `unix.Pselect` → `pselect6`（sigmask 非 nil 时）
- 任何直接 `unix.Syscall(SYS_RT_SIG*)` 且传 sigset 的代码

**关键事实**：新旧世界的**系统调用编号、信号编号完全一致**（SIGPROF=27、
SIG_BLOCK=0 等，均已对照龙芯 abi1.0 工具链 runtime 源码确认），**唯一差异
是 sigset 的大小**（旧世界 16 字节 vs 新世界 8 字节）。因此补丁只需改
`_C__NSIG` 一个常量。

## 4. 本项目的解决方案（已内置）

`build_all.sh` 会自动构建两种 loong64 产物：

| 产物目录 | ABI | 工具链 | 适用系统 |
|---|---|---|---|
| `release/linux-loong64` | 新世界 OBJ-v1 | 上游 Go | 麒麟 V11、openEuler、Loongnix 25 等 |
| `release/linux-loong64-abi1` | 旧世界 OBJ-v0 | 龙芯 abi1.0 Go | 麒麟 V10、UOS V20、Loongnix 20 |

打包产物对应 `bbx-*-linux-loong64.tar.gz`（新世界）与
`bbx-*-linux-loong64-abi1.tar.gz` / `*.loongarch64-abi1.rpm` /
`*.loong64-abi1.deb`（旧世界）。`upgrade.sh` 会通过动态加载器路径自动
识别目标系统所属世界并选择对应包（可用 `BBX_ARCH=loong64-abi1` 强制指定）。

### 4.1 构建机准备（一次性）

```bash
# 1. 下载龙芯 abi1.0 工具链（amd64 版可在 x86 构建机上交叉编译）
#    版本目录见 http://ftp.loongnix.cn/toolchain/golang/
curl -LO http://ftp.loongnix.cn/toolchain/golang/go-1.25/abi1.0/go1.25.11.linux-amd64.tar.gz

# 2. 解压到默认位置（可用 LOONGSON_GO 环境变量指定其它位置）
mkdir -p ~/toolchains/go-abi1.0
tar -C ~/toolchains/go-abi1.0 --strip-components=1 -xzf go1.25.11.linux-amd64.tar.gz
~/toolchains/go-abi1.0/bin/go version   # go version go1.25.11 linux/amd64

# 3. 正常执行构建即可，build_all.sh 检测到工具链后自动追加 abi1 产物
./build_all.sh
```

### 4.2 build_all.sh 旧世界构建做了什么

1. **用龙芯 abi1.0 工具链编译**（`LOONGSON_GO`，默认
   `~/toolchains/go-abi1.0/bin/go`）——修补 Go runtime 自身的信号 ABI；
2. **就地补丁 x/sys**——在 abi1 专用独立模块缓存
   （`ABI1_GOMODCACHE`，默认 `~/gopath-abi1/pkg/mod`，不污染正常构建）
   里把 `_C__NSIG = 0x41` 改为 `0x80`，使 `PthreadSigmask` / `Pselect` /
   `Signalfd` 的 sigsetsize 变为 16。
   - 为什么不用 `go build -overlay`：**Go 禁止 overlay 替换模块缓存内的文件**；
   - 为什么不用龙芯源（`goproxy.loongnix.cn:3000`）的魔改模块：该源需要
     关闭校验并重新生成 go.sum，且构建机网络不一定能访问。自行补丁一个
     常量即可达到同样效果；
   - 模块缓存目录默认只读，补丁前需 `chmod -R u+w`（脚本已处理）；
   - 补丁已幂等，重复构建安全；
3. **exporter 采集器 recover 兜底**（`exporter/exporter_linux.go` 的
   `safeFactory`）：任何采集器 panic（如 ebpf 在不兼容内核/ABI 上）降级为
   日志告警并跳过，不再拖垮整个服务。

可调环境变量：

| 变量 | 默认值 | 说明 |
|---|---|---|
| `LOONGSON_GO` | `~/toolchains/go-abi1.0/bin/go` | 龙芯 abi1.0 工具链路径 |
| `ABI1_GOPROXY` | `${GOPROXY:-https://goproxy.cn,direct}` | 模块代理 |
| `ABI1_GOMODCACHE` | `~/gopath-abi1/pkg/mod` | 独立模块缓存（补丁打在这里） |

## 5. 排查手册：新增依赖/其它程序遇到同类问题

### 5.1 症状 → 原因速查

| 症状 | 大概率原因 |
|---|---|
| 启动即段错误，无任何输出 | 二进制是错的那个世界（`readelf -h` 确认 e_flags） |
| `panic: masking profiler signal: invalid argument` | x/sys sigsetsize 未打补丁（见 4.2 第 2 点） |
| 某个系统调用返回 `EINVAL` / `ENOSYS` | 该调用在新旧世界的编号或参数结构不同（少见，见 5.3） |
| `zsh: 没有那个文件或目录: ./foo`（文件确实存在） | 动态链接的异世界程序，ELF interpreter 不存在 |

### 5.2 引入新依赖前的审计清单

凡是「关心系统底层 ABI」的依赖都要检查。在模块缓存里 grep：

```bash
# 直接发系统调用的（重点看信号相关）
grep -rn "SYS_RT_SIG\|PthreadSigmask\|sigprocmask" ~/gopath-abi1/pkg/mod/<依赖>/

# 用 unix.Syscall/RawSyscall 的
grep -rn "unix.RawSyscall\|unix.Syscall(" ~/gopath-abi1/pkg/mod/<依赖>/ | grep -v _test
```

判断原则：

- **纯 /proc、/sys 读取**（如 prometheus/procfs、gopsutil 大部分）→ 安全；
- **ioctl / netlink / socket 选项** → 系统调用编号两界一致，安全；
- **信号掩码相关 syscall**（rt_sig* 系列、signalfd、pselect6/ppoll 带
  sigmask）→ 会踩 sigsetsize 的坑，走 x/sys 的已被 4.2 的补丁覆盖；
  绕过 x/sys 自己硬编码的需单独处理；
- **clone3** → 旧世界内核没有，凡用它必须有 ENOSYS 回落（Go runtime 用
  clone，不受影响）；
- **stat 系列** → 旧世界内核反而齐全（fstat/newfstatat 都有），安全。

### 5.3 确认某个 ABI 细节差异的权威方法

不要猜，直接对比**龙芯 abi1.0 工具链源码**（它就是旧世界 ABI 的权威参考，
构建机上已解压到 `~/toolchains/go-abi1.0/src/`）：

```bash
# 例如确认旧世界的某个信号常量 / 结构体 / syscall 编号
grep -rn "_SIGPROF\|_SIG_BLOCK" ~/toolchains/go-abi1.0/src/runtime/defs_linux_loong64.go
grep -rn "sigset" ~/toolchains/go-abi1.0/src/runtime/os_linux_loong64.go
# 与上游对比
diff <(cat ~/toolchains/go-abi1.0/src/runtime/os_linux_loong64.go) \
     <(go env GOROOT)/src/runtime/os_linux_loong64.go
```

辅助参考（社区整理的新旧世界 ABI 细节）：

- 旧世界与新世界（概念与判断）：https://areweloongyet.com/docs/old-and-new-worlds/
- 底层细节（syscall/信号/glibc 差异）：https://areweloongyet.com/docs/world-compat-details/

### 5.4 验证补丁真的编进了二进制

业务二进制用 `-ldflags "-s -w"`  strip 了符号，无法直接 objdump。
用一个**不 strip 的小程序**引用同一函数来验证：

```bash
mkdir /tmp/sigtest && cd /tmp/sigtest
cat > main.go <<'EOF'
package main

import (
	"fmt"
	"golang.org/x/sys/unix"
)

func main() {
	var set unix.Sigset_t
	set.Val[0] = 1 << 26
	fmt.Println(unix.PthreadSigmask(unix.SIG_BLOCK, &set, nil))
}
EOF
go mod init sigtest && go mod tidy
GOOS=linux GOARCH=loong64 CGO_ENABLED=0 \
  GOMODCACHE=~/gopath-abi1/pkg/mod \
  ~/toolchains/go-abi1.0/bin/go build -o sigtest .
~/toolchains/go-abi1.0/bin/go tool objdump -s "main.main" sigtest | grep rtSigprocmask -B3
# 补丁生效：CALL ...rtSigprocmask 前的第 4 参数是 MOVW $16, R7（未补丁为 $8）
```

### 5.5 如果能访问龙芯源

龙芯官方 goproxy（`http://goproxy.loongnix.cn:3000`）里的 x/sys 等模块已是
旧世界魔改版，可替代手工补丁：

```bash
# 龙芯源模块与上游同名同版本但内容不同，go.sum 校验必然失败，需重新生成
GOPROXY=http://goproxy.loongnix.cn:3000 GOSUMDB=off go mod tidy
# 构建时使用独立 GOMODCACHE，防止魔改模块污染正常构建
ABI1_GOPROXY=http://goproxy.loongnix.cn:3000 ./build_all.sh
```

注意：新世界构建**不可**使用龙芯源（魔改模块对新世界反而是坏的）。

## 6. 已知适配记录（vigil/bbx）

| 日期 | 问题 | 处理 |
|---|---|---|
| 2026-08-13 | 麒麟 V10 上 bbx-server 启动即段错误 | 确认新世界二进制跑旧世界内核；build_all.sh 增加 abi1.0 工具链构建目标（commit `816b9a1`） |
| 2026-08-14 | cilium/ebpf `masking profiler signal: invalid argument` panic | x/sys `_C__NSIG` 0x41→0x80 就地补丁；exporter 采集器加 recover 兜底（commit `8f6e798`） |
