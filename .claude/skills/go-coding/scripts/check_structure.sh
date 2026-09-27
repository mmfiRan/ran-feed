#!/usr/bin/env bash
# ran-feed 结构静态检查 —— 分层依赖方向 入口点隔离 跨服务契约 包名禁令
set -euo pipefail

# 自带定位仓库根 从任意目录调用都能跑
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
cd "$REPO_ROOT"

# 按域规则目前只有 content 域达标。某个域迁移完成后，把它的根路径加进这一行即可放开，
# 全仓规则（下面 1~7 条）各域都已达标，不分域。
STRUCT_SCOPES="app/rpc/content"

VIOLATIONS=0

# 打印一条违规 参数：说明、命中行
report() {
  echo "❌ $1"
  echo "$2"
  VIOLATIONS=$((VIOLATIONS + 1))
}

# ── 全仓规则 ────────────────────────────────────────────────────

# 1 一个服务不得 import 另一个服务的 internal
CROSS_INTERNAL=""
for svc in app/rpc/*/ app/front app/admin; do
  svc="${svc%/}"
  hit=$(grep -rn --include='*.go' "\"ran-feed/${svc}/internal/" app 2>/dev/null | grep -v '_test.go' | grep -v "^${svc}/" || true)
  if [ -n "$hit" ]; then
    CROSS_INTERNAL="${CROSS_INTERNAL}${hit}
"
  fi
done
if [ -n "$CROSS_INTERNAL" ]; then
  report "存在跨服务 internal import：" "$CROSS_INTERNAL"
fi

# 2 Repository 依赖只能向下 不引 Logic 或 Component
REPO_UP=$(grep -rn --include='*.go' -E '"ran-feed/app/[a-z]+/[a-z0-9]+/internal/(logic|common/component)' app/*/*/internal/repositories 2>/dev/null | grep -v '_test.go' || true)
if [ -n "$REPO_UP" ]; then
  report "Repository 层引用了 Logic 或 Component：" "$REPO_UP"
fi

# 3 Logic 只经 Repository 接口包 不引实现子包
LOGIC_IMPL=$(grep -rn --include='*.go' -E '"ran-feed/app/[a-z]+/[a-z0-9]+/internal/repositories/' app/*/*/internal/logic 2>/dev/null | grep -v '_test.go' || true)
if [ -n "$LOGIC_IMPL" ]; then
  report "Logic 直接引用了 Repository 实现子包：" "$LOGIC_IMPL"
fi

# 4 pkg 不得反向依赖 app
PKG_APP=$(grep -rn --include='*.go' -E '"ran-feed/app/' pkg 2>/dev/null || true)
if [ -n "$PKG_APP" ]; then
  report "pkg 反向引用了 app：" "$PKG_APP"
fi

# 5 跨服务 Redis Key 只有一个契约来源 服务内不得再写字面量
DUP_KEY=$(grep -rn --include='*.go' -E '"(feed:bigv:global|feed:hot:dirty)"' app pkg 2>/dev/null | grep -v '^pkg/rediskey/' | grep -v '_test.go' || true)
if [ -n "$DUP_KEY" ]; then
  report "跨服务 Redis Key 在服务内重复定义（应走 pkg/rediskey）：" "$DUP_KEY"
fi

# 6 internal/common 根目录不放 Go 文件
COMMON_ROOT=$(find app/*/*/internal/common -maxdepth 1 -name '*.go' 2>/dev/null || true)
if [ -n "$COMMON_ROOT" ]; then
  report "internal/common 根目录出现 Go 文件：" "$COMMON_ROOT"
fi

# 7 common/component 下不放兜底子目录
FALLBACK_DIR=$(find app/*/*/internal/common/component -maxdepth 1 -type d \( -name utils -o -name helper -o -name common \) 2>/dev/null || true)
if [ -n "$FALLBACK_DIR" ]; then
  report "common/component 下出现兜底子目录：" "$FALLBACK_DIR"
fi

# ── 按域规则 ────────────────────────────────────────────────────
for svc in $STRUCT_SCOPES; do
  # 两个入口点禁止互相调用
  hit=$(grep -rn --include='*.go' -E "\"ran-feed/${svc}/internal/(logic|mq/consumer)" "${svc}/internal/cron" 2>/dev/null | grep -v '_test.go' || true)
  if [ -n "$hit" ]; then
    report "${svc} 的 cron 引用了 Logic 或 Consumer：" "$hit"
  fi

  # Consumer 不直接承载数据访问 应经 Component
  hit=$(grep -rn --include='*.go' -E "\"ran-feed/${svc}/internal/repositories" "${svc}/internal/mq/consumer" 2>/dev/null | grep -v '_test.go' || true)
  if [ -n "$hit" ]; then
    report "${svc} 的 consumer 引用了业务 Repository：" "$hit"
  fi

  # Logic 只保显式依赖 不存完整 ServiceContext
  hit=$(grep -rn --include='*_logic.go' 'svcCtx \*svc.ServiceContext' "${svc}/internal/logic" 2>/dev/null | grep -v 'func New' || true)
  if [ -n "$hit" ]; then
    report "${svc} 的 Logic 保存了完整 ServiceContext：" "$hit"
  fi

  # 无 utils 兜底目录
  hit=$(find "${svc}" -type d -name utils 2>/dev/null || true)
  if [ -n "$hit" ]; then
    report "${svc} 下出现 utils 兜底目录：" "$hit"
  fi

  # 枚举一致性测试存在
  if [ ! -f "${svc}/internal/common/enums/enum_consistency_test.go" ]; then
    report "${svc} 缺少枚举一致性测试：" "${svc}/internal/common/enums/enum_consistency_test.go"
  fi
done

# ── 结论 ────────────────────────────────────────────────────────
if [ "$VIOLATIONS" -eq 0 ]; then
  echo "    结构静态检查通过 ✓（全仓规则 + 按域：${STRUCT_SCOPES}）"
  exit 0
fi

echo "❌ 结构静态检查未通过 共 ${VIOLATIONS} 条 规则见 .claude/skills/go-coding/references/structure.md"
exit 1
