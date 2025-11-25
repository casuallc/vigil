#!/bin/bash

set -e  # 出错立即退出

# 检查必要目录是否存在
for dir in conf scripts/appctl.sh release/linux-amd64 release/linux-arm64; do
    if [ ! -e "$dir" ]; then
        echo "错误: $dir 不存在，请检查项目结构。"
        exit 1
    fi
done

# 清理旧输出
rm -rf release/vigil-*.tar.gz
rm -rf release/vigil

# 打包通用函数
package_arch() {
    local arch=$1          # amd64 或 arm64
    local src_dir=$2       # 二进制源目录，如 release/linux-amd64
    local output_name=$3   # 输出文件名，如 vigil-linux-amd64.tar.gz

    echo "📦 打包 $arch 架构..."

    local temp_pkg="release/vigil"  # 统一使用 vigil 作为目录名

    # 清理临时目录
    rm -rf "$temp_pkg"

    # 创建目录结构
    mkdir -p "$temp_pkg/bin" "$temp_pkg/logs"

    # 拷贝配置和脚本
    cp -r conf "$temp_pkg/"
    cp scripts/appctl.sh "$temp_pkg/bin/"
    chmod +x "$temp_pkg/bin/appctl.sh"

    # 拷贝对应架构的二进制
    cp "$src_dir"/vigil-dev* "$temp_pkg/vigil"
    cp "$src_dir"/vigil-cli-dev* "$temp_pkg/vigil-cli"
    chmod +x "$temp_pkg/vigil"
    chmod +x "$temp_pkg/vigil-cli"

    # 打包（在 dest 目录内打包）
    (cd release && tar -zcvf "$output_name" vigil)

    # 清理临时目录（避免影响下一个架构）
    rm -rf "$temp_pkg"

    echo "✅ $output_name 生成完成"
}

# 分别打包 amd64 和 arm64
package_arch "amd64" "release/linux-amd64" "vigil-linux-amd64.tar.gz"
package_arch "arm64" "release/linux-arm64" "vigil-linux-arm64.tar.gz"

echo "🎉 所有架构打包完成！"