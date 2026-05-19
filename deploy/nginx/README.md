# Nginx 部署配置

## 文件结构

```
deploy/nginx/
├── nginx.conf              # 主配置：limit_req_zone、log_format、server_tokens 等
├── conf.d/
│   ├── default.conf        # HTTP 入口（80 端口），默认启用
│   └── ssl.conf.example    # HTTPS 模板（443 端口），默认禁用
└── certs/                  # 证书目录（内容不入库，挂载到容器 /etc/nginx/certs）
    ├── .gitignore          # 忽略所有证书文件
    └── README.md           # 本文件
```

## 安全特性（sec-002）

启用后默认提供：

| 项 | 状态 | 来源 |
|---|---|---|
| `X-Content-Type-Options: nosniff` | 启用 | default.conf / ssl.conf.example |
| `X-Frame-Options: SAMEORIGIN` | 启用 | 同上 |
| `Referrer-Policy: strict-origin-when-cross-origin` | 启用 | 同上 |
| `Permissions-Policy` | 启用（禁用 geolocation/microphone/camera） | 同上 |
| `Content-Security-Policy-Report-Only` | 启用（观察模式） | 同上 |
| `Strict-Transport-Security` | **仅 HTTPS** | ssl.conf.example |
| `server_tokens off` | 启用 | nginx.conf |
| 全局 API 限流 10 r/s burst=20 | 启用 | `/v1/` location |
| 登录/注册限流 1 r/s burst=5 | 启用 | `/v1/login`、`/v1/users` location |

## 启用 HTTPS

### 1. 准备证书

**生产环境**：上传由 CA 签发的证书：

```bash
cp /path/to/your.crt  deploy/nginx/certs/server.crt
cp /path/to/your.key  deploy/nginx/certs/server.key
chmod 600 deploy/nginx/certs/server.key
```

**本地开发**：用 openssl 生成自签证书（浏览器会警告，仅供测试）：

```bash
openssl req -x509 -nodes -newkey rsa:2048 \
  -keyout deploy/nginx/certs/server.key \
  -out    deploy/nginx/certs/server.crt \
  -days 365 \
  -subj "/CN=localhost"
```

### 2. 切换配置

```bash
cd deploy/nginx/conf.d
mv default.conf default.conf.disabled   # 让出 80 端口给 ssl.conf 的 301 重定向
cp ssl.conf.example ssl.conf            # 启用 HTTPS server 块
```

### 3. 解除 docker-compose 配置

打开 `deploy/docker-compose.yml`，在 `nginx` 服务下解除：

- `${NGINX_HTTPS_PORT}:443` 端口映射
- `./nginx/certs:/etc/nginx/certs:ro` 卷挂载

确认 `.env` 中 `NGINX_HTTPS_PORT=443` 存在。

### 4. 重启容器

```bash
cd deploy
docker compose up -d nginx
```

### 5. 验证

```bash
# 80 端口应 301 到 https
curl -I http://localhost/v1/login
# 443 端口应正常响应，且带 HSTS / 其他安全头
curl -kI https://localhost/v1/login | grep -iE "strict-transport|frame-options|content-type-options"
```

## 关闭 HTTPS（回滚）

```bash
cd deploy/nginx/conf.d
rm ssl.conf
mv default.conf.disabled default.conf
docker compose restart nginx
```

## CSP Report-Only 升级到强制模式

当前 `Content-Security-Policy-Report-Only` 仅记录违规，不拦截。
前端审计资源引用后，将响应头改为 `Content-Security-Policy` 即可强制生效。
建议先观察 1~2 周浏览器 devtools console 是否有违规日志。

## 与应用层限频的关系

| 层 | 维度 | 触发 | 实现 |
|---|---|---|---|
| Nginx（sec-002） | 客户端 IP | 503 | `limit_req_zone $binary_remote_addr` |
| user-rpc（sec-001） | 手机号 | 业务错误 | Redis ZSET 滑动窗口 |

两层互补：网关挡持续撞击、应用挡精确账号枚举。