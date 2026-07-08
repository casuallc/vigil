#!/usr/bin/env bash
set -euo pipefail

# Nexus 配置
NEXUS_URL="https://nexus.apusic.com/service/rest/v1/components?repository=file-server"
NEXUS_USER="devops_test"
NEXUS_PASS="Cloud_dev123"
NEXUS_DIR="/admq/bbx"

# Git Bash / MSYS2 下会把以 / 开头的参数自动转成 Windows 路径
# （例如 /admq/bbx -> C:/Program Files/Git/admq/bbx），导致 Nexus 收到的 directory 不对。
# 禁用该自动路径转换。
export MSYS2_NO_PATHCONV=1

# 获取版本号（优先环境变量，其次 git tag，最后默认 1.0.0）
VERSION="${VERSION:-$(git describe --tags --exact-match 2>/dev/null || echo '1.0.0')}"

echo "🚀 推送包到 Nexus: ${NEXUS_DIR} (版本: ${VERSION})"

# 切换到项目根目录
cd "$(dirname "$0")"

# 定义要推送的包文件
PKG_FILES=(
    "release/bbx-${VERSION}-linux-amd64.tar.gz"
    "release/bbx-${VERSION}-linux-arm64.tar.gz"
    "release/bbx-${VERSION}.x86_64.rpm"
    "release/bbx-${VERSION}.aarch64.rpm"
    "release/bbx-${VERSION}.amd64.deb"
    "release/bbx-${VERSION}.arm64.deb"
)

uploaded=0
skipped=0

# 遍历并上传
for pkg in "${PKG_FILES[@]}"; do
    if [ ! -f "$pkg" ]; then
        echo "⚠️  跳过不存在: $pkg"
        ((skipped++)) || true
        continue
    fi

    filename=$(basename "$pkg")
    echo -n "📤 上传: $filename ... "

    http_code=$(curl -s -o /dev/null -w "%{http_code}" \
        -F "raw.asset1=@${pkg}" \
        -u "${NEXUS_USER}:${NEXUS_PASS}" \
        -F "raw.directory=${NEXUS_DIR}" \
        -F "raw.asset1.filename=${filename}" \
        "$NEXUS_URL")

    if [ "$http_code" = "200" ] || [ "$http_code" = "204" ] || [ "$http_code" = "201" ]; then
        echo "✅ 成功"
        ((uploaded++)) || true
    else
        echo "❌ 失败 (HTTP ${http_code})"
    fi
done

echo ""
echo "🎉 推送完成: ${uploaded} 个成功, ${skipped} 个跳过"
