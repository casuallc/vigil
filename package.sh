#!/bin/bash

set -e  # 出错立即退出

VERSION="${VERSION:-$(git describe --tags --exact-match 2>/dev/null || echo '1.0.0')}"

# 检查必要目录是否存在
for dir in conf scripts/appctl.sh release/linux-amd64 release/linux-arm64; do
    if [ ! -e "$dir" ]; then
        echo "错误: $dir 不存在，请检查项目结构。"
        exit 1
    fi
done

# 清理旧输出
rm -rf release/bbx-*.tar.gz
rm -rf release/bbx-*.rpm
rm -rf release/bbx

# 打包通用函数
package_arch() {
    local arch=$1          # amd64 或 arm64
    local src_dir=$2       # 二进制源目录，如 release/linux-amd64
    local output_name=$3   # 输出文件名，如 bbx-linux-amd64.tar.gz

    echo "📦 打包 $arch 架构..."

    local temp_pkg="release/bbx"  # 统一使用 bbx 作为目录名

    # 清理临时目录
    rm -rf "$temp_pkg"

    # 创建目录结构
    mkdir -p "$temp_pkg/bin" "$temp_pkg/logs"

    # 拷贝配置和脚本
    cp upgrade.sh "$temp_pkg/"
    cp LICENSE "$temp_pkg/"
    cp -r conf "$temp_pkg/"
    cp scripts/appctl.sh "$temp_pkg/bin/"
    chmod +x "$temp_pkg/bin/appctl.sh"

    # 拷贝对应架构的二进制
    cp "$src_dir"/bbx-server* "$temp_pkg/bbx-server"
    cp "$src_dir"/bbx-cli-* "$temp_pkg/bbx-cli"
    chmod +x "$temp_pkg/bbx-server" "$temp_pkg/bbx-cli"

    # 打包 tar.gz（在 dest 目录内打包）
    (cd release && tar -zcvf "$output_name" bbx)
    echo "✅ $output_name 生成完成"

    # 使用 nfpm 打包 RPM
    if command -v nfpm >/dev/null 2>&1; then
        local rpm_arch
        if [ "$arch" = "amd64" ]; then
            rpm_arch="x86_64"
        else
            rpm_arch="aarch64"
        fi

        # 生成临时 nfpm 配置
        sed -e "s|{{ARCH}}|$arch|g" \
            -e "s|{{VERSION}}|$VERSION|g" \
            nfpm.template.yaml > "/tmp/nfpm-$arch.yaml"

        nfpm package -f "/tmp/nfpm-$arch.yaml" -p rpm -t "release/bbx-${VERSION}.${rpm_arch}.rpm"
        rm -f "/tmp/nfpm-$arch.yaml"
        echo "✅ bbx-${VERSION}.${rpm_arch}.rpm 生成完成"
    else
        echo "⚠️  nfpm 未安装，跳过 RPM 打包"
        echo "    安装命令: go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest"
    fi

    # 清理临时目录（避免影响下一个架构）
    rm -rf "$temp_pkg"
}

# 分别打包 amd64 和 arm64
package_arch "amd64" "release/linux-amd64" "bbx-linux-amd64.tar.gz"
package_arch "arm64" "release/linux-arm64" "bbx-linux-arm64.tar.gz"

echo "🎉 所有架构打包完成！"
