# RAN·FEED

[中文](/README.md) | [English](/README_EN.md)

![RAN·FEED banner](docs/images/banner.png)

**内容/信息流后台系统**，覆盖发布、互动与推荐/关注流等关键体验场景

![license](https://img.shields.io/badge/license-MIT-blue.svg)
![go](https://img.shields.io/badge/go-1.25+-00ADD8.svg)
![status](https://img.shields.io/badge/status-active-success.svg)

---

![Star the repo](docs/images/star-badge.svg)

## 目录
- [功能概览](#功能概览)
- [技术栈](#技术栈)
- [项目展示](#项目展示)
- [技术架构](#技术架构)
- [业务架构](#业务架构)
- [目录结构](#目录结构)
- [快速开始](#快速开始)
- [配置说明](#配置说明)
- [部署说明](#部署说明)
- [后续开发计划](#后续开发计划)
- [License](#license)

---

## 功能概览
- **用户体系**：注册、登录、登出、个人主页与资料获取
- **个人资料**：头像上传、昵称/简介/性别等基础信息
- **内容发布**：文章/视频发布、删除、内容详情
- **互动能力**：点赞/取消、收藏/取消、评论/删除、回复评论
- **关系能力**：关注/取关、关注与粉丝状态
- **内容流**：推荐流、关注流、用户发布列表、用户收藏列表
- **计数统计**：点赞/收藏/评论计数与用户获赞/被收藏统计

---

## 技术栈
- Go
- go-zero
- MySQL
- Redis
- Kafka
- Canal
- XXL-Job
- ELK
- Prometheus
- Grafana
- Jaeger
- OpenTelemetry
- Nginx
- Docker Compose

---

## 项目展示
### 演示（前台/业务侧）
| 场景 | 截图 |
| --- | --- |
| 推荐流 | ![推荐流](docs/images/recommand-feed.png) |
| 关注流 | ![关注流](docs/images/follow-feed.png) |
| 内容详情 / 评论互动 | ![内容详情](docs/images/content-detail.png) |
| 发布内容（图文/视频） | ![发布内容](docs/images/publish-content.png) |
| 发布文章 | ![发布文章](docs/images/publish-article.png) |
| 发布视频 | ![发布视频](docs/images/publish-video.png) |

### 基建与观测（运维/平台侧）
| 模块 | 截图 |
| --- | --- |
| XXL-Job | ![XXL-Job](docs/images/cron.png) |
| Jaeger Tracing | ![Jaeger](docs/images/jaeger.png) |
| Grafana Dashboard | ![Grafana](docs/images/grafana.png) |
| Kibana / ELK | ![Kibana](docs/images/kibana.png) |

---

## 技术架构
![技术架构](docs/images/tech-architecture.png)

---

## 业务架构
![业务架构](docs/images/business-architecture.png)

---

## 目录结构
```
app/                # 服务代码
  front/            # HTTP API
    etc/            # 服务配置
    internal/       # 业务实现
  rpc/              # RPC 服务
    content/        # 内容域
    interaction/    # 互动域
    user/           # 用户域
    count/          # 计数域
build/              # Dockerfile
deploy/             # docker-compose 与部署配置
pkg/                # 公共库
script/             # SQL与启动脚本
```

---

## 快速开始
### 1. 环境准备
- 推荐环境：**Ubuntu 22.04**（建议在该环境下使用 Docker Compose 一键启动）
- Docker + Docker Compose
- Go (本地开发)

### 注意事项
- 请在 `.env` 中补充 OSS 相关配置：
  - `OSS_PROVIDER`：云厂商（如 `aliyun`）
  - `OSS_REGION`：地域（如 `cn-beijing`）
  - `OSS_BUCKET_NAME`：Bucket 名称
  - `OSS_ACCESS_KEY_ID` / `OSS_ACCESS_KEY_SECRET`：访问密钥
  - `OSS_ENDPOINT`：上传 Endpoint
  - `OSS_UPLOAD_DIR`：上传目录前缀
  - `OSS_PUBLIC_HOST`：公网访问域名
  - `OSS_ROLE_ARN`：可选，使用 RAM 角色时填写

### 2. 一键启动（推荐）
```bash
./script/start.sh
```
**启动完成后访问：`http://localhost`**

### 3. 停止
```bash
./script/stop.sh
```

---

## 配置说明
配置文件全部位于 `app/**/etc/*.yaml`

环境变量通过 `${VAR}` 方式注入，启动时会自动加载：
- 本地：根目录 `.env`
- 容器：`deploy/.env`

建议先补齐以下关键配置：
- 数据源：`MYSQL_HOST` / `REDIS_HOST` / `ETCD_HOST` / `KAFKA_BROKERS`
- 观测：`OTEL_ENDPOINT` / `LOG_PATH` / `PROM_HOST`
- OSS：`OSS_PROVIDER` / `OSS_REGION` / `OSS_BUCKET_NAME` / `OSS_ACCESS_KEY_ID` / `OSS_ACCESS_KEY_SECRET` / `OSS_ENDPOINT` / `OSS_UPLOAD_DIR` / `OSS_PUBLIC_HOST`

示例（容器场景）：
```
MYSQL_HOST=mysql
REDIS_HOST=redis
ETCD_HOST=etcd
KAFKA_BROKERS=kafka:9092
OTEL_ENDPOINT=otel-collector:4317
LOG_PATH=/var/log/ran-feed
PROM_HOST=0.0.0.0
OSS_PROVIDER=aliyun
OSS_REGION=cn-beijing
OSS_BUCKET_NAME=your-bucket
OSS_ACCESS_KEY_ID=your-key-id
OSS_ACCESS_KEY_SECRET=your-key-secret
OSS_ENDPOINT=https://oss-cn-beijing.aliyuncs.com
OSS_UPLOAD_DIR=uploads
OSS_PUBLIC_HOST=https://your-bucket.oss-cn-beijing.aliyuncs.com
```

---

## 部署说明
推荐使用 Docker Compose（`deploy/docker-compose.yml`）

基础流程：
1. 进入 `deploy/`，准备 `.env`（可从根目录 `.env` 复制并按需修改）。
2. 启动：
```bash
cd deploy
docker compose --env-file .env up -d --build
```
3. 验证：
```bash
docker compose ps
```
4. 停止：
```bash
docker compose down
```

---

## 后续开发计划
- 推荐/关注流的基础完善与优化
- 评论与互动提醒
- 个人主页内容聚合与收藏/发布列表完善
- 热门内容与榜单逻辑优化
- 搜索能力（内容/用户）
- IM/私信能力（基础会话/消息）

---

## License
MIT
