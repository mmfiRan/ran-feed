# proto 编写规范

新增或修改 `.proto` 时遵守。适用于 `app/rpc/**/proto/*.proto` 与 `pkg/commonpb/common.proto`，
定义与风格对齐 [Protocol Buffers 官方 Style Guide](https://protobuf.dev/programming-guides/style/)。
改 `.proto` 属升级处理先与用户确认；改完用 goctl 重生成（命令见
[new-interface.md](new-interface.md)），绝不手改 `*.pb.go` / `*_grpc.pb.go` / client。

## package / go_package

- `package` 小写点分、全仓唯一，带项目前缀 `ranfeed`（proto 不允许连字符，不用 `ran-feed`）：`package ranfeed.count;`
  - **不加 `v1`**：内部微服务无多版本兼容需求，加版本号徒增 gRPC 全名长度与生成目录层级
- `go_package` 写**完整模块路径** `ran-feed/app/rpc/<svc>`，不用相对路径（`./count` 是反例）
- 跨服务公共 message（如 `EnumValue`）进 `pkg/commonpb`，由各服务 proto `import "pkg/commonpb/common.proto"` 引用，不重复定义
- import 顺序固定：`syntax → package → imports → option`，标准库 import（`google/protobuf/*.proto`）同样在 option 之前

```proto
syntax = "proto3";

package ranfeed.admin;

import "pkg/commonpb/common.proto";
import "google/protobuf/empty.proto";
import "google/protobuf/timestamp.proto";

option go_package = "ran-feed/app/rpc/admin";
```

## 枚举

- 值一律 `UPPER_SNAKE_CASE` 且**带枚举类型名前缀**：`BizType` 的值写 `BIZ_TYPE_LIKE`，不写裸 `LIKE`（撞名风险）
- 0 值（第一个值）用 `*_UNSPECIFIED`，不用 `*_UNKNOWN`
- 嵌套在 message 内且需被多处引用的枚举**提升为顶级**并整体更名，值前缀同步
  （`ContentUploadsCredentialsReq` 的嵌套 `Scene` → 顶级 `UploadScene`，值 `UPLOAD_SCENE_*`）
- proto3 分号可选，与 goctl 生成风格一致，不强制补分号
- Go 生成常量规则为 `<EnumType>_<VALUE>`：`BIZ_TYPE_LIKE` → `BizType_BIZ_TYPE_LIKE`，Go 侧引用同步该命名

```proto
// BizType 计数业务类型
enum BizType {
  BIZ_TYPE_UNSPECIFIED = 0;
  BIZ_TYPE_LIKE = 10;
  BIZ_TYPE_FAVORITE = 20;
}
```

## 时间字段

- 「真实时间」字段一律 `google.protobuf.Timestamp`（`int64` 毫秒时间戳是反例），并 `import "google/protobuf/timestamp.proto"`
  - Go 侧赋值 `timestamppb.Now()` / `timestamppb.New(t)`、读取 `.AsTime()`，不用 `UnixMilli()`
- 保留 `int64` 的例外（非时间语义，不改）：
  - 分页游标 `cursor` / `next_cursor`
  - 复合排序键（`cursor_updated_at` / `cursor_id` / `next_cursor_updated_at` / `next_cursor_id`）
  - ES external version 语义字段（如 `ContentIndexItem.version`）

## 空响应

- rpc 无返回内容时统一返回 `google.protobuf.Empty`，不自定义空 message（`message IncRes {}` 是反例），
  并 `import "google/protobuf/empty.proto"`

```proto
rpc Inc(IncReq) returns (google.protobuf.Empty);
```

## 结构与命名

- 定义顺序：message / enum 在前，`service` 收尾；两个 service 之间不夹定义
- 字段跳号显式声明 `reserved N;`；顶级定义之间空一行
- message / rpc 方法 PascalCase，请求 / 响应固定 `XxxReq` / `XxxRes` 后缀，字段 snake_case
- `total` 字段统一 `int64`；`page` / `page_size` 保持 `uint32`（页码不会溢出 32 位）
- 每个 message / 字段 / 枚举 / rpc 都补 `//` 注释（生成后即 Go 侧 doc 注释）

## package 改名的影响面

`package` 加前缀后 gRPC service 全名变长（如 `ranfeed.count.CounterService`）：
检查拦截器 / 监控埋点 / 日志是否有硬编码 service 全名；etcd 服务发现走 go-zero 配置层服务名
（如 `count.rpc`），与 proto `package` 无关，不受影响。
