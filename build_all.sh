#!/bin/bash

# ========================================
# Ultimate Go Multi-command Cross-platform Build Script
# Features:
#   - Support building specific commands: ./build_all.sh cmd1 cmd2
#   - Auto-scan cmd/* (when no args)
#   - UPX compression (optional)
#   - Build for all platforms
#   - Generate unified release.zip + version notes
# Environment: Windows Git Bash / Linux / macOS
# ========================================

# ---------- Config ----------
PROJECT_NAME="${PROJECT_NAME:-apusic}"
CMD_DIR="cmd"
BUILD_DIR="pkg"
RELEASE_DIR="release"
BUILD_TIME=$(date "+%Y-%m-%d %H:%M:%S")
GIT_COMMIT=$(git rev-parse --short=8 HEAD 2>/dev/null || echo "unknown")
GIT_TAG=$(git describe --tags --exact-match 2>/dev/null || echo "1.0")
VERSION="${GIT_TAG}"

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

log()     { echo -e "${BLUE}📦 $*${NC}"; }
success() { echo -e "${GREEN}✅ $*${NC}"; }
warn()    { echo -e "${YELLOW}⚠️  $*${NC}"; }
error()   { echo -e "${RED}❌ $*${NC}"; }

has_upx() { command -v upx >/dev/null 2>&1 && upx --version >/dev/null 2>&1; }

# ---------- Main Function ----------
main() {
    log "Starting build system"
    log "Project: $PROJECT_NAME | Version: $VERSION"

    # ========== Parse CLI Args ==========
    if [ $# -eq 0 ]; then
        # Auto-scan cmd/*
        if [ ! -d "$CMD_DIR" ]; then
            error "Directory '$CMD_DIR' does not exist, and no commands specified"
            exit 1
        fi

        # Compatible with old macOS bash (no mapfile)
        if command -v mapfile >/dev/null 2>&1; then
            mapfile -t CMD_LIST < <(find "$CMD_DIR" -mindepth 1 -maxdepth 1 -type d -exec basename {} \; | sort)
        else
            CMD_LIST=()
            while IFS= read -r dir; do
                CMD_LIST+=("$(basename "$dir")")
            done < <(find "$CMD_DIR" -mindepth 1 -maxdepth 1 -type d | sort)
        fi

        if [ ${#CMD_LIST[@]} -eq 0 ]; then
            error "No command directories found under '$CMD_DIR'"
            exit 1
        fi
        log "Auto-discovered commands: ${CMD_LIST[*]}"
    else
        # Use specified args
        CMD_LIST=("$@")
        log "Building specified commands: ${CMD_LIST[*]}"
    fi

    # Validate each command directory
    for CMD in "${CMD_LIST[@]}"; do
        if [ ! -d "$CMD_DIR/$CMD" ]; then
            error "Command directory '$CMD_DIR/$CMD' does not exist!"
            exit 1
        fi
        if ! find "$CMD_DIR/$CMD" -name "*.go" | grep -q "."; then
            error "No Go source files found in '$CMD'"
            exit 1
        fi
    done

    # ========== Generate BPF bindings if needed ==========
    # bpf2go requires both clang and llvm-strip; skip if either is missing
    # so the committed generated files are used instead.
    if command -v clang >/dev/null 2>&1 && command -v llvm-strip >/dev/null 2>&1; then
        log "Regenerating BPF objects..."
        go generate ./exporter/bpf/gen.go || warn "BPF code generation failed"
    else
        warn "clang or llvm-strip not found, skipping BPF code generation"
    fi

    # ========== Clean & Prepare ==========
    rm -rf "$BUILD_DIR" "$RELEASE_DIR"
    mkdir -p "$BUILD_DIR" "$RELEASE_DIR"

    # ========== UPX Detection ==========
    if has_upx; then
        UPX_AVAILABLE=1
        success "UPX enabled"
    else
        UPX_AVAILABLE=0
        warn "UPX not installed, skipping compression"
    fi

    # ========== Check zip availability ==========
    if ! command -v zip >/dev/null 2>&1; then
        warn "zip command not found. Will skip release zip creation."
        SKIP_ZIP=1
    else
        SKIP_ZIP=0
    fi

    # ========== Target Platforms ==========
    TARGETS=(
        # "windows amd64 1"
        # "windows arm64 1"
        "linux   amd64   0"
        "linux   arm64   0"
        "linux   loong64 0"
        # "darwin  amd64 0"
        # "darwin  arm64 0"
    )

    # ========== Build Each Command ==========
    for CMD in "${CMD_LIST[@]}"; do
        echo
        CMD_PATH="$CMD_DIR/$CMD"
        log "========== Building command: $CMD | Path: $CMD_PATH =========="

        for TARGET in "${TARGETS[@]}"; do
            set -- $TARGET
            GOOS=$1
            GOARCH=$2
            IS_WINDOWS=$3

            OUTPUT_NAME="$CMD-$VERSION-$GOOS-$GOARCH"
            BINARY_NAME="$OUTPUT_NAME"
            [ $IS_WINDOWS -eq 1 ] && BINARY_NAME="$OUTPUT_NAME.exe"

            OUTPUT_PATH="$BUILD_DIR/$BINARY_NAME"

            log "→ $GOOS/$GOARCH"

            export GOOS GOARCH CGO_ENABLED=0

            go build \
                -ldflags "-s -w \
                -X 'github.com/casuallc/vigil/version.Version=$VERSION' \
                -X 'github.com/casuallc/vigil/version.BuildTime=$BUILD_TIME' \
                -X 'github.com/casuallc/vigil/version.GitCommit=$GIT_COMMIT' \
                -X 'github.com/casuallc/vigil/version.GitBranch=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")'" \
                -o "$OUTPUT_PATH" \
                "./$CMD_PATH"

            if [ $? -ne 0 ]; then
                error "Build failed: $CMD ($GOOS/$GOARCH)"
                exit 1
            fi
            success "✔ $BINARY_NAME"

            # UPX Compression (skip for darwin and loong64)
            if [ $UPX_AVAILABLE -eq 1 ] && [ "$GOOS" != "darwin" ] && [ "$GOARCH" != "loong64" ]; then
                upx --best --quiet "$OUTPUT_PATH"
                COMPRESSED=$(upx -q -l "$OUTPUT_PATH" | tail -1 | awk '{print $6}')
                success "⚡ Compressed: $COMPRESSED"
            fi
        done
    done

    # ========== Old-world (ABI1.0) loong64 build ==========
    # 龙芯 LoongArch 分「旧世界」(麒麟 V10 / Loongnix 20 / UOS V20，内核 4.19/5.4/5.10)
    # 与「新世界」(上游 ABI2.0，内核 ≥5.19) 两个互不兼容的 ABI。
    # 上游 Go 编出的 loong64 二进制是新世界，在旧世界系统上启动即段错误
    # （runtime 初始化时 rt_sigprocmask 因 NSIG 不一致失败，Go 故意崩溃）。
    # 旧世界必须使用龙芯 abi1.0 工具链构建：
    #   http://ftp.loongnix.cn/toolchain/golang/go-1.25/abi1.0/go1.25.11.linux-amd64.tar.gz
    # 可通过环境变量覆盖：
    #   LOONGSON_GO    abi1.0 工具链的 go 可执行文件路径
    #                  上游模块即可：新旧世界绝大多数系统调用编号一致，信号 ABI
    #                  差异中 runtime 部分由工具链修补，x/sys 部分由下方的就地
    #                  补丁处理（打在 ABI1_GOMODCACHE 独立缓存里）。若能访问龙芯源
    #                  也可设为 http://goproxy.loongnix.cn:3000，但需先执行
    #                  GOSUMDB=off go mod tidy 重新生成 go.sum（龙芯源模块为
    #                  魔改版，与上游校验和不一致）。
    #   ABI1_GOMODCACHE 旧世界构建使用的独立模块缓存（x/sys 旧世界补丁就地打在
    #                  这里，避免污染正常构建的模块缓存）
    LOONGSON_GO="${LOONGSON_GO:-$HOME/toolchains/go-abi1.0/bin/go}"
    if [ -x "$LOONGSON_GO" ]; then
        log "========== Building old-world (ABI1.0) linux/loong64 =========="
        ABI1_ENV=(GOOS=linux GOARCH=loong64 CGO_ENABLED=0
                  "GOPROXY=${ABI1_GOPROXY:-${GOPROXY:-https://goproxy.cn,direct}}"
                  "GOMODCACHE=${ABI1_GOMODCACHE:-$HOME/gopath-abi1/pkg/mod}")

        # ========== Patch x/sys for the old-world signal ABI ==========
        # 旧世界内核 NSIG=128，rt_sigprocmask / signalfd4 / pselect6 等调用
        # 要求 sigsetsize=16；上游 x/sys 对 loong64 硬编码 _C__NSIG=0x41
        # （新世界，size=8），导致 cilium/ebpf 的 PthreadSigmask 直接 EINVAL
        # panic（龙芯 abi1.0 工具链只修补了 Go runtime，管不到第三方模块）。
        # 这里在 abi1 专用的独立模块缓存里把 x/sys 的 _C__NSIG 改为 0x80
        # （0x80/8=16，PthreadSigmask/Pselect/Signalfd 的 size 随之变为 16）。
        # 信号编号新旧世界一致（SIGPROF=27），无需其它改动。
        # 注意：go build -overlay 不允许替换模块缓存内的文件，故直接就地补丁；
        # 独立的 ABI1_GOMODCACHE 保证不会污染正常构建的模块缓存。
        env "${ABI1_ENV[@]}" "$LOONGSON_GO" mod download golang.org/x/sys || {
            error "下载 golang.org/x/sys 失败"
            exit 1
        }
        XSYS_DIR=$(env "${ABI1_ENV[@]}" "$LOONGSON_GO" list -m -f '{{.Dir}}' golang.org/x/sys 2>/dev/null)
        if [ -n "$XSYS_DIR" ] && [ -f "$XSYS_DIR/unix/ztypes_linux_loong64.go" ]; then
            # 模块缓存的目录和文件默认只读，sed -i 需要在同目录创建临时文件
            chmod -R u+w "$XSYS_DIR" 2>/dev/null
            sed -i 's/const _C__NSIG = 0x41/const _C__NSIG = 0x80/' \
                "$XSYS_DIR/unix/ztypes_linux_loong64.go"
            if ! grep -q "const _C__NSIG = 0x80" "$XSYS_DIR/unix/ztypes_linux_loong64.go"; then
                error "生成 x/sys 旧世界补丁失败：$XSYS_DIR 中未找到 _C__NSIG = 0x41"
                error "x/sys 版本可能已变更，请检查 ztypes_linux_loong64.go"
                exit 1
            fi
            log "Applied x/sys old-world signal ABI patch (sigsetsize 8→16)"
        else
            warn "无法定位 golang.org/x/sys 模块目录，未应用旧世界信号补丁"
            warn "cilium/ebpf 在旧世界内核上会 panic（exporter 已有 recover 兜底）"
        fi

        mkdir -p "$RELEASE_DIR/linux-loong64-abi1"
        for CMD in "${CMD_LIST[@]}"; do
            OUTPUT_NAME="$CMD-$VERSION-linux-loong64-abi1"
            OUTPUT_PATH="$BUILD_DIR/$OUTPUT_NAME"

            log "→ linux/loong64-abi1"

            env "${ABI1_ENV[@]}" "$LOONGSON_GO" build \
                -ldflags "-s -w \
                -X 'github.com/casuallc/vigil/version.Version=$VERSION' \
                -X 'github.com/casuallc/vigil/version.BuildTime=$BUILD_TIME' \
                -X 'github.com/casuallc/vigil/version.GitCommit=$GIT_COMMIT' \
                -X 'github.com/casuallc/vigil/version.GitBranch=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")'" \
                -o "$OUTPUT_PATH" \
                "./$CMD_DIR/$CMD"

            if [ $? -ne 0 ]; then
                error "Build failed: $CMD (linux/loong64-abi1)"
                exit 1
            fi
            success "✔ $OUTPUT_NAME"

            cp "$OUTPUT_PATH" "$RELEASE_DIR/linux-loong64-abi1/" || {
                error "Copy failed: $OUTPUT_NAME"
                exit 1
            }
        done
    else
        warn "未找到龙芯 abi1.0 工具链 ($LOONGSON_GO)，跳过旧世界 loong64 构建"
        warn "下载地址: http://ftp.loongnix.cn/toolchain/golang/go-1.25/abi1.0/"
    fi

    # ========== Create Unified Release ==========
    log "📦 Creating unified release package..."

    # Create per-platform directories
    for TARGET in "${TARGETS[@]}"; do
        set -- $TARGET
        GOOS=$1
        GOARCH=$2
        IS_WINDOWS=$3

        PLATFORM_DIR="$RELEASE_DIR/$GOOS-$GOARCH"
        mkdir -p "$PLATFORM_DIR"

        for CMD in "${CMD_LIST[@]}"; do
            OUTPUT_NAME="$CMD-$VERSION-$GOOS-$GOARCH"
            BINARY_NAME="$OUTPUT_NAME"
            [ $IS_WINDOWS -eq 1 ] && BINARY_NAME="$OUTPUT_NAME.exe"

            cp "$BUILD_DIR/$BINARY_NAME" "$PLATFORM_DIR/" || {
                error "Copy failed: $BINARY_NAME"
                exit 1
            }
        done
    done

    # Add version file
    cat > "$RELEASE_DIR/VERSION.txt" << EOF
Project: $PROJECT_NAME
Version: $VERSION
Git Commit: $GIT_COMMIT
Build Time: $BUILD_TIME
Commands: ${CMD_LIST[*]}
Built on: $(uname -s)
EOF

    # Add release notes
    cat > "$RELEASE_DIR/RELEASE_NOTES.md" << EOF
# Release: $VERSION

- **Project**: $PROJECT_NAME
- **Build Time**: $BUILD_TIME
- **Git Commit**: \`$GIT_COMMIT\`
- **Commands**: ${CMD_LIST[*]}

## File List

EOF

    find "$RELEASE_DIR" -type f -not -name "RELEASE_NOTES.md" | sort | while read file; do
        echo "- \$(basename "$file")" >> "$RELEASE_DIR/RELEASE_NOTES.md"
    done

    # Package into release.zip
    if [ $SKIP_ZIP -eq 0 ]; then
        RELEASE_ZIP="$PROJECT_NAME-release-$VERSION.zip"
        (cd "$RELEASE_DIR" && zip -rq "../$BUILD_DIR/$RELEASE_ZIP" .)
        success "🎉 Unified release package created: ./$BUILD_DIR/$RELEASE_ZIP"
    fi
    success "Release content located in: ./$RELEASE_DIR/"

    # List all build artifacts
    echo
    success "✅ All builds completed! Output files:"
    ls -lh "$BUILD_DIR/" | grep -E "\.(exe|zip|gz)$" | awk '{print "  " $9}'

    echo
    success "💡 Usage examples:"
    success "  ./build_all.sh                    # Build all commands"
    success "  ./build_all.sh server agent       # Build only server and agent"
}

# ---------- Entry ----------
main "$@"
