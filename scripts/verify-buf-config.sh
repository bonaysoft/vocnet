#!/bin/bash

# Buf 配置验证脚本

echo "🔍 验证 Buf 配置..."

# 检查必要的文件是否存在
echo "📁 检查配置文件..."
required_files=(
    "buf.gen.yaml" 
    "buf.work.yaml"
    "api/proto/buf.yaml"
)

for file in "${required_files[@]}"; do
    if [[ -f "$file" ]]; then
        echo "✅ $file 存在"
    else
        echo "❌ $file 缺失"
        exit 1
    fi
done

# 检查 proto 文件是否存在
echo "📋 检查 protobuf 文件..."
proto_count=$(find api/proto -name "*.proto" | wc -l)
echo "📄 找到 $proto_count 个 protobuf 文件"

if [[ $proto_count -eq 0 ]]; then
    echo "❌ 没有找到 protobuf 文件"
    exit 1
fi

# 检查生成目录
echo "📂 检查生成目录..."
mkdir -p api/gen api/openapi

echo "✅ Buf 配置验证完成！"
echo ""
echo "🚀 下一步："
echo "1. 运行 'make install-tools' 安装 buf"
echo "2. 运行 'make buf-deps' 更新依赖"  
echo "3. 运行 'make generate' 生成代码"