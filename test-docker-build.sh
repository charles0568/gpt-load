#!/bin/bash

# Docker 構建測試腳本
# 用於在本地測試多平台 Docker 構建

set -e

echo "🐳 開始 Docker 構建測試..."

# 獲取版本號
VERSION=$(grep 'var Version = ' internal/version/version.go | cut -d'"' -f2)
echo "📦 版本號: $VERSION"

echo "🔧 設置 Docker Buildx..."
docker buildx create --name gpt-load-builder --use --bootstrap 2>/dev/null || docker buildx use gpt-load-builder

echo "🏗️  測試構建 (僅 linux/amd64)..."
docker buildx build \
  --platform linux/amd64 \
  --build-arg VERSION=$VERSION \
  --tag charles0568/gpt-load:$VERSION-test \
  --load \
  .

echo "✅ 單平台構建成功！"

echo "🚀 測試運行容器..."
CONTAINER_ID=$(docker run -d -p 3001:3001 charles0568/gpt-load:$VERSION-test)

echo "⏳ 等待服務啟動..."
sleep 10

echo "🔍 檢查健康狀態..."
if curl -f http://localhost:3001/health >/dev/null 2>&1; then
    echo "✅ 健康檢查通過！"
else
    echo "❌ 健康檢查失敗"
    docker logs $CONTAINER_ID
fi

echo "🧹 清理測試環境..."
docker stop $CONTAINER_ID >/dev/null 2>&1 || true
docker rm $CONTAINER_ID >/dev/null 2>&1 || true
docker rmi charles0568/gpt-load:$VERSION-test >/dev/null 2>&1 || true

echo "🎉 Docker 構建測試完成！"
