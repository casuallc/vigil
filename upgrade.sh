#!/bin/bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

RESTART=false

# 解析参数
while [[ $# -gt 0 ]]; do
    case $1 in
        --restart)
            RESTART=true
            shift
            ;;
        *)
            echo "❌ 未知参数: $1"
            echo "用法: $0 [--restart]"
            exit 1
            ;;
    esac
done

echo "🚀 开始执行智能更新流程..."

# === 自动检测架构 ===
ARCH=""
case "$(uname -m)" in
    x86_64|amd64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    *)
        echo "❌ 不支持的 CPU 架构: $(uname -m)"
        exit 1
        ;;
esac

echo "🔧 检测到系统架构: $ARCH"

# === 动态构造预期文件名 ===
VERSION="1.0"
OS="linux"

NEW_CLI="bbx-cli-${VERSION}-${OS}-${ARCH}"
NEW_SRV="bbx-server-${VERSION}-${OS}-${ARCH}"
CURRENT_CLI="bbx-cli"
CURRENT_SRV="bbx-server"

# 创建备份目录
BACKUP_DIR="backups"
mkdir -p "$BACKUP_DIR"
BACKUP_SUFFIX="$(date +%Y%m%d_%H%M%S)"

# 获取 MD5
get_md5() {
    if command -v md5sum >/dev/null 2>&1; then
        md5sum "$1" | cut -d' ' -f1
    elif command -v md5 >/dev/null 2>&1; then
        md5 -r "$1" | cut -d' ' -f1
    else
        echo "⚠️  无法计算 MD5" >&2
        return 1
    fi
}

# 初始化更新标志
updated_any=false

# ===== 处理 CLI =====
if [ -f "$NEW_CLI" ]; then
    echo "🔍 发现 CLI 新版本: $NEW_CLI"
    need_update_cli=false
    if [ ! -f "$CURRENT_CLI" ] || [ "$(get_md5 "$NEW_CLI")" != "$(get_md5 "$CURRENT_CLI")" ]; then
        need_update_cli=true
        echo "🔄 bbx-cli 需要更新"
    else
        echo "✅ bbx-cli 无变化，跳过"
    fi

    if [ "$need_update_cli" = true ]; then
        # 备份并更新
        if [ -f "$CURRENT_CLI" ]; then
            backup_name="${CURRENT_CLI}.${BACKUP_SUFFIX}"
            echo "📦 备份 bbx-cli -> $BACKUP_DIR/$backup_name"
            cp "$CURRENT_CLI" "$BACKUP_DIR/$backup_name"
        fi
        cp "$NEW_CLI" "$CURRENT_CLI"
        chmod +x "$CURRENT_CLI"
        updated_any=true

        # 清理临时文件
        rm -f "$NEW_CLI"
        echo "🧹 已清理 CLI 临时文件"
    else
        # 无变化，但仍可清理（可选）
        rm -f "$NEW_CLI"
        echo "🧹 CLI 无变化，已清理临时文件"
    fi
else
    echo "ℹ️  未提供 CLI 新版本（$NEW_CLI 不存在），跳过"
fi

# ===== 处理 Server =====
if [ -f "$NEW_SRV" ]; then
    echo "🔍 发现 Server 新版本: $NEW_SRV"
    need_update_srv=false
    if [ ! -f "$CURRENT_SRV" ] || [ "$(get_md5 "$NEW_SRV")" != "$(get_md5 "$CURRENT_SRV")" ]; then
        need_update_srv=true
        echo "🔄 bbx-server 需要更新"
    else
        echo "✅ bbx-server 无变化，跳过"
    fi

    if [ "$need_update_srv" = true ]; then
        # 停止服务（如果尚未停止且需要重启）
        if [ "$RESTART" = true ] && [ "$updated_any" = false ]; then
            echo "🛑 正在停止服务..."
            if ! ./bin/appctl.sh stop; then
                echo "⚠️  停止服务失败，但继续更新文件..."
            fi
        fi

        # 备份并更新
        if [ -f "$CURRENT_SRV" ]; then
            backup_name="${CURRENT_SRV}.${BACKUP_SUFFIX}"
            echo "📦 备份 bbx-server -> $BACKUP_DIR/$backup_name"
            cp "$CURRENT_SRV" "$BACKUP_DIR/$backup_name"
        fi
        cp "$NEW_SRV" "$CURRENT_SRV"
        chmod +x "$CURRENT_SRV"
        updated_any=true

        # 清理临时文件
        rm -f "$NEW_SRV"
        echo "🧹 已清理 Server 临时文件"
    else
        rm -f "$NEW_SRV"
        echo "🧹 Server 无变化，已清理临时文件"
    fi
else
    echo "ℹ️  未提供 Server 新版本（$NEW_SRV 不存在），跳过"
fi

# ===== 最终处理 =====
if [ "$updated_any" = false ]; then
    echo "💤 无任何文件需要更新"
    if [ "$RESTART" = true ]; then
        echo "🔁 用户要求重启，正在重启服务..."
        ./bin/appctl.sh restart
        echo "✅ 服务已重启（无文件变更）"
    else
        echo "ℹ️  未重启（默认行为）"
    fi
else
    # 启动服务（如果启用了重启）
    if [ "$RESTART" = true ]; then
        echo "🔁 用户要求重启，正在重启服务..."
        ./bin/appctl.sh restart
        echo "✅ 更新并重启完成！"
    else
        echo "✅ 文件已按需更新（未重启服务）"
        echo "💡 如需生效，请手动执行: ./bin/appctl.sh restart"
    fi
fi