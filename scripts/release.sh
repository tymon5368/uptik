#!/usr/bin/env bash
set -e

# Usage: ./scripts/release.sh [version]
# Example: ./scripts/release.sh 1.0.0 or ./scripts/release.sh v1.0.0

VERSION_INPUT=$1

if [ -z "$VERSION_INPUT" ]; then
  echo "❌ Error: Vui lòng cung cấp số phiên bản. Ví dụ: ./scripts/release.sh 1.0.0"
  exit 1
fi

# Remove leading 'v' if provided
VERSION="${VERSION_INPUT#v}"
TAG="v${VERSION}"

echo "=================================================="
echo "🚀 Chuẩn bị phát hành UpTik phiên bản: ${TAG}"
echo "=================================================="

# 1. Cập nhật version trong version.go
sed -i "s/Version = \".*\"/Version = \"${VERSION}\"/" version.go
echo "✅ Đã cập nhật version.go -> ${VERSION}"

# 2. Cập nhật version trong wails.json
sed -i "s/\"productVersion\": \".*\"/\"productVersion\": \"${VERSION}\"/" wails.json
echo "✅ Đã cập nhật wails.json -> ${VERSION}"

# 3. Cập nhật version trong frontend/package.json và App.svelte
sed -i "s/\"version\": \".*\"/\"version\": \"${VERSION}\"/" frontend/package.json
sed -i "s/let appVersion = \$state<string>('.*');/let appVersion = \$state<string>('${VERSION}');/" frontend/src/App.svelte
echo "✅ Đã cập nhật frontend/package.json & App.svelte -> ${VERSION}"

# 4. Chạy kiểm thử Frontend & TypeScript
echo "🔍 Kiểm tra chất lượng frontend..."
(cd frontend && bun run check && bun run build)
echo "✅ Frontend typecheck & build thành công!"

# 5. Chạy kiểm thử Go Backend
echo "🔍 Kiểm tra Go backend tests..."
go test -tags "webkit2_41" ./...
echo "✅ Backend test suite 100% passed!"

# 6. Git commit và tạo Tag
git add version.go wails.json frontend/package.json frontend/src/App.svelte
if ! git diff --cached --quiet; then
  git commit -m "chore(release): bump version to ${TAG}"
fi

# 7. Gắn git tag
if git rev-parse "$TAG" >/dev/null 2>&1; then
  echo "⚠️ Tag $TAG đã tồn tại. Đang cập nhật tag..."
  git tag -d "$TAG"
fi

git tag -a "$TAG" -m "Release ${TAG}"
echo "🏷️ Đã tạo git tag: ${TAG}"

echo "=================================================="
echo "🎉 Hoàn tất chuẩn bị release ${TAG}!"
echo "👉 Để kích hoạt GitHub Actions tự động build đa nền tảng, chạy lệnh:"
echo "   git push origin main --tags"
echo "=================================================="
