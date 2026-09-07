# proto 编写规范

`.proto` 是 RPC 接口与传输类型的源定义，生成 Go 代码前先对齐 Protocol Buffers 官方 Style Guide。
完整改造对照表见 `proto-refactor-plan.md`，本文是编码时的速查。

## package / go_package

- `package` 统一加项目前缀 `ranfeed`（proto 不允许连字符，故不用 `ran-feed`），如 `ranfeed.admin`
- `go_package` 用完整模块路径，如 `ran-feed/app/rpc/admin/admin`（保留现有 `<svc>/<svc>` 双层目录，不拍平）

用完整模块路径，protoc 才能配合 `--go_opt=module=ran-feed` 把生成文件放到正确位置（命令见
[new-interface.md](new-interface.md)）。

## import 顺序

`import` 放 `option` 之前，顺序是 `syntax → package → imports → option`：

```proto
package ranfeed.admin;

import "pkg/commonpb/common.proto";

option go_package = "ran-feed/app/rpc/admin/admin";
```

## 枚举命名

枚举值 `UPPER_SNAKE_CASE` 且带类型名前缀，0 值用 `*_UNSPECIFIED`：

```proto
enum BizType {
  BIZ_TYPE_UNSPECIFIED = 0;
  BIZ_TYPE_LIKE = 1;
}
```

- 加类型名前缀是因为 proto 的枚举值在同一 package 内是平铺的，不加前缀容易撞名（不同枚举都叫 `LIKE` 会冲突）
- 0 值用 `UNSPECIFIED`（未指定）而非 `UNKNOWN`，这是官方 Style Guide 约定
- 嵌套在 message 里的枚举提升为顶级（`ContentUploadsCredentialsReq` 内的 `Scene` → 顶级 `UploadScene`）

Go 侧生成的常量名是 `<EnumType>_<VALUE>`，如 `BizType_BIZ_TYPE_LIKE`。改枚举值后全仓库替换旧常量。

## 空 message → google.protobuf.Empty

删除自定义空 message，rpc 返回 `google.protobuf.Empty`：

```proto
import "google/protobuf/empty.proto";
rpc Inc(IncReq) returns (google.protobuf.Empty);
```

## 时间字段 → google.protobuf.Timestamp

真实时间字段用 `google.protobuf.Timestamp`，不用 `int64` 毫秒时间戳：

```proto
import "google/protobuf/timestamp.proto";
message ContentDetail {
  google.protobuf.Timestamp published_at = 1;
}
```

- Go 侧类型从 `int64` 变为 `*timestamppb.Timestamp`，赋值用 `timestamppb.Now()`，读取用 `.AsTime()`
- 不是时间语义的字段不改：分页游标（cursor）、ES external version（version）保留 `int64`

## 定义顺序

message / enum 定义统一放文件顶部，service 收尾。不要把定义插在两个 service 之间。

## reserved

字段跳号要显式声明 `reserved`，防止未来误用已废弃的字段号：

```proto
message DeleteCommentReq {
  reserved 4;
  int64 user_id = 1;
}
```

## total 类型

计数 `total` 字段统一 `int64`；页码 `page` / `page_size` 保持 `uint32`（页码不会溢出 32 位）。

## 注释

每个 message、字段补 `//` 注释，说明语义。