---
name: git-commit
description: 为本项目生成 git 提交信息时遵循的规范 — Conventional Commits 格式 type scope subject body 一次提交只做一个功能 提交前通过编译/vet/测试 不带任何 AI 标识。在 commit 前应用。
---

# ran-feed Git 提交规范

遵循 [Conventional Commits](https://www.conventionalcommits.org/zh-hans/)。
**提交描述精简、一句话概括，不写详细描述，不带任何 AI 标识。**

## 提交格式

**默认只写一行 subject**：

```
<type>(<scope>): <subject>
```

仅当改动重大且"为什么"不显而易见时，才加简短 body（一两行）：

```
<type>(<scope>): <subject>

<一两行说明 why 可引用功能条目 ID>
```

## Type（必填，一次提交只选一个）

| Type | 用途 | 示例 |
|------|------|------|
| `feat` | 新功能 | `feat(feed): 实现推拉结合 fan-out` |
| `fix` | Bug 修复 | `fix(user): 登录禁用账号校验` |
| `refactor` | 重构（不改变可观察行为） | `refactor(content): 提取 afterPublish` |
| `perf` | 性能优化 | `perf(comment): 评论列表加 Redis 缓存` |
| `docs` | 仅文档变更 | `docs: 完善设计模式枚举` |
| `test` | 测试相关 | `test(jwt): 补充 token 解析单测` |
| `chore` | 杂项（依赖、构建脚本、配置） | `chore: 升级 go-zero 到 v1.7` |
| `style` | 仅格式变更（不改逻辑） | `style: gofmt 全量格式化` |
| `ci` | CI/CD 配置变更 | `ci: 增加 vet 检查步骤` |

## Scope（推荐）

用服务名（`user` / `content` / `interaction` / `count` / `front`）或模块名
（`feed` / `auth` / `oss` / `comment`），让 `git log --grep="(feed)"` 这类检索有效。

## Subject（必填）

- 中英文都可以，但**同一仓库保持一致**（本项目用中文）
- ≤ 50 字符
- 动词开头，不加句号
- 描述"做了什么"，不重复 Type 信息

## Body（默认不写）

- **默认只有一行 subject 不写 body**，保证 git log 简洁
- 仅当改动重大且"为什么这么改"不显而易见时才加，控制在一两行
- body 只解释 **为什么**（diff 已显示 what），不复述改动，每行 ≤ 72 字符

示例（绝大多数提交就这一行）：

```
fix(content): Redis 操作移出 MySQL 事务
```

## 反例

```text
❌ "update"                              # 无信息
❌ "feat: add some changes"              # 模糊
❌ "fix bug"                             # 没说什么 bug
❌ "feat: 功能A + 功能B 一起提交"         # 多功能混合提交
❌ "Updated XXX (no test, no review)"    # 工作日志而非变更说明
```

## 与流程联动

- **一次提交只做一个功能**，与"一次只做一件事"的节奏对齐
- **提交前必须通过编译 / vet / 测试**，失败不准 commit
- 写 body 时才在 body 引用对应功能条目 ID
- 本地存在 `.harness/` 目录时，以其中内部流程为准（任务条目、验证脚本）
- 永远不用 `--no-verify` 跳过 git hooks，hook 报错先修问题
- **绝不在提交信息中加入任何 AI 生成标识或 Co-Authored-By**
