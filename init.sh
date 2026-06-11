#!/usr/bin/env bash
# ran-feed Harness 初始化与验证脚本
# 每次新会话开始时运行，确保环境健康后再开始编码
set -euo pipefail

echo "=== ran-feed Harness 初始化 ==="
echo ""

# ── 1. 检查 Go 工具链 ──────────────────────────────────────────
echo "[1/5] 检查 Go 工具链..."
if ! command -v go &>/dev/null; then
  echo "❌ 未找到 go 命令，请先安装 Go 1.25+"
  exit 1
fi
GO_VERSION=$(go version | awk '{print $3}')
echo "    Go 版本：$GO_VERSION ✓"

# ── 2. 编译所有服务 ──────────────────────────────────────────────
echo ""
echo "[2/5] 编译所有服务（go build ./...）..."
# 确保 go.sum 完整（首次克隆或切换分支后可能缺少条目）
go mod download -x 2>/dev/null || true
if go build ./... 2>&1; then
  echo "    编译通过 ✓"
else
  echo "❌ 编译失败，请先修复编译错误再继续"
  exit 1
fi

# ── 3. 静态分析 ──────────────────────────────────────────────────
echo ""
echo "[3/5] 静态分析（go vet ./...）..."
if go vet ./... 2>&1; then
  echo "    静态分析通过 ✓"
else
  echo "❌ go vet 报告问题，请先修复"
  exit 1
fi

# ── 4. 检查测试（如果存在） ──────────────────────────────────────
echo ""
echo "[4/5] 运行测试..."
TEST_COUNT=$(find . -name "*_test.go" -not -path "./.git/*" | wc -l | tr -d ' ')
if [ "$TEST_COUNT" -eq 0 ]; then
  echo "    ⚠️  暂无测试文件（feat-013 未完成），跳过"
else
  echo "    发现 $TEST_COUNT 个测试文件，执行 go test ./..."
  if go test ./... 2>&1; then
    echo "    测试通过 ✓"
  else
    echo "❌ 测试失败，请先修复"
    exit 1
  fi
fi

# ── 5. 显示当前功能状态摘要 ──────────────────────────────────────
echo ""
echo "[5/5] 功能状态摘要..."
if command -v jq &>/dev/null && [ -f "feature_list.json" ]; then
  DONE=$(jq '[.features[] | select(.status=="done")] | length' feature_list.json)
  IN_PROG=$(jq '[.features[] | select(.status=="in-progress")] | length' feature_list.json)
  NOT_STARTED=$(jq '[.features[] | select(.status=="not-started")] | length' feature_list.json)
  TOTAL=$(jq '.features | length' feature_list.json)
  echo "    总计：$TOTAL 个功能/修复"
  echo "    已完成：$DONE  |  进行中：$IN_PROG  |  未开始：$NOT_STARTED"
else
  echo "    （安装 jq 可显示统计，或直接查看 feature_list.json）"
fi

echo ""
echo "=== 验证完成 ✓ ==="
echo ""
echo "下一步："
echo "  1. 阅读 feature_list.json，选取一个 not-started 或 in-progress 的条目"
echo "  2. 只做一个功能/修复，完成后更新 progress.md 和 feature_list.json"
echo "  3. 修复完成后再次运行 ./init.sh 确认验证通过"
echo ""
echo "提示：P0/P1 历史 Bug 已全部修复，剩余待办为新功能 feat-014~018（not-started）"
echo ""
echo "规则提醒：每次会话前请完整阅读 CLAUDE.md"