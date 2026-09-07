# 业务枚举（Go 层）

DB 枚举字段的 Go 层规范：业务枚举是 DB 映射 + 合法校验 + 中文描述的真相源。
proto 层的枚举命名见 [proto-style.md](proto-style.md)，规范全文见 `.harness/ENUM-RULES.md`。

## 基础设施（pkg/enums）

基础设施在 `pkg/enums`：

- `Int` 约束：限定枚举底层是整数类型
- `Enum` 接口：`Int32()` / `Valid()` / `Message()` / `String()`，业务枚举统一实现它
- `EnumValue{code,name,message}`：对外响应的统一结构
- `Value` / `Parse` / `MustParse`：枚举与 int32 的安全互转

通用枚举（跨表通用，如 `IsDeleted`）放 `pkg/enums`；业务枚举留在各域 `internal/common/enums/`，不进 `pkg`——避免 `pkg` 膨胀成业务知识库。

## 模板

```go
// Package enums <域> 服务级业务枚举 实现 pkg/enums.Enum 契约
package enums

type XxxStatusEnum int32

const (
    XxxStatusUnknown  XxxStatusEnum = 0
    XxxStatusEnabled  XxxStatusEnum = 10
    XxxStatusDisabled XxxStatusEnum = 20
)

var xxxStatusNames    = map[XxxStatusEnum]string{...} // 英文 name
var xxxStatusMessages = map[XxxStatusEnum]string{...} // 中文 message

func (s XxxStatusEnum) Int32() int32   { return int32(s) }
func (s XxxStatusEnum) Valid() bool    { _, ok := xxxStatusNames[s]; return ok }
func (s XxxStatusEnum) String() string { return xxxStatusNames[s] }
func (s XxxStatusEnum) Message() string { return xxxStatusMessages[s] }
// 可选业务语义方法 如 IsActive() bool
```

## 使用方法（分层职责）

各层对枚举各管一段，越界就重复维护了：

| 层 | 职责 |
|----|------|
| proto | `enum` 定义（请求强类型）+ `commonpb.EnumValue`（响应，跨域通用） |
| RPC logic | pb 枚举 ↔ 业务枚举转换；响应组装 `EnumValue`（用 `pkg/enums/commonpb.go` 的 `ToCommonPB` 一行搞定） |
| BFF | 薄转换：请求 int32 → pb 枚举；响应透传 `EnumValue`，不重复维护中文 message |
| 前端 | 收 int32 code，直接展示 `EnumValue.message` |

使用时的几个约定：

- **DB 持久化字段不自造字面量**：`status` / `visibility` / `content_type` / `is_deleted` 这类字段，一律用业务枚举或 pb 枚举，不要写 `ContentStatusPublished int32 = 30` 这种重复定义——值一改两处漂移。
- **从 DB 行转枚举**：直接 `content.ContentStatus(row.Status)`；需要拦非法值时用 `Parse` 校验 `Valid`。
- **HTTP 入参约定值不算 DB 枚举**：如 `action=takedown/restore`、第三方标识 `aliyun`，这类不是 DB 字段，按「单服务专属」入该服务 `common/consts` 即可，不强制走 pb。