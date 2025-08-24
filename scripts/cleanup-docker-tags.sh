#!/bin/bash

# Docker 標籤清理腳本
# 用於清理 Docker Hub 上的舊標籤，只保留 latest 和最新的幾個版本

echo "🧹 開始清理 Docker 標籤..."

# 設置變量
DOCKER_REPO="charles0568/gpt-load"
KEEP_TAGS=("latest" "v1.3.5" "v1.3.4" "v1.3.3")

echo "📋 保留的標籤："
for tag in "${KEEP_TAGS[@]}"; do
    echo "  - $tag"
done

echo ""
echo "⚠️  注意：此腳本需要手動在 Docker Hub 上刪除不需要的標籤"
echo "🌐 請訪問：https://hub.docker.com/r/$DOCKER_REPO/tags"
echo ""
echo "建議刪除的標籤："
echo "  - main-c7c45a2"
echo "  - latest-zh-tw"
echo "  - zh-tw"
echo "  - main"
echo "  - main-d2f999f"
echo ""
echo "✅ 保留 latest 標籤作為主要使用標籤"
echo "✅ 保留最新的幾個版本標籤用於回滾"
echo ""
echo "🔧 如果需要自動化清理，可以使用 Docker Hub API 或 GitHub Actions"
