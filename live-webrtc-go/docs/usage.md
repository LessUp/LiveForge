# 使用指南

本文覆盖本地开发、容器部署与常见排障技巧，帮助你快速体验 live-webrtc-go。

## 前置要求

- Go 1.21 及以上，`go` 命令需在 `PATH` 中。
- （可选）Docker / Docker Compose，用于容器化部署。
- 浏览器需支持 WebRTC，并允许访问摄像头/麦克风。

## 本地开发

### 方式一：使用启动脚本（推荐）

```bash
# 进入仓库根目录
./scripts/start.sh
```

脚本特点：

- 自动创建 `records`、`.gocache`、`.gomodcache` 目录，并把 `GOCACHE`/`GOMODCACHE` 指向仓库内，避免污染全局。
- 默认执行 `go mod tidy`（通过 `SKIP_TIDY=1 ./scripts/start.sh` 可跳过）。
- 支持在 `.env.local` 中定义环境变量，例如：

```bash
# .env.local 示例
HTTP_ADDR=:9090
ALLOWED_ORIGIN=https://example.com
AUTH_TOKEN=demo-token
```

### 方式二：手动执行

```bash
GOCACHE=$(pwd)/.gocache GO111MODULE=on go run ./cmd/server
```

若首次拉取代码，需要先运行 `go mod tidy` 以安装依赖。

服务启动后可访问：

- 推流：`http://localhost:8080/web/publisher.html`
- 播放：`http://localhost:8080/web/player.html`
- 房间列表：`http://localhost:8080/api/rooms`
- 录制文件：`http://localhost:8080/api/records`
- 指标：`http://localhost:8080/metrics`

## Docker / Compose

### 构建镜像

```bash
docker build -t live-webrtc-go:latest .
```

### 直接运行容器

```bash
docker run --rm -p 8080:8080 \
  -e RECORD_ENABLED=1 \
  -e RECORD_DIR=/records \
  -v "$PWD/records:/records" \
  live-webrtc-go:latest
```

### 使用 docker-compose

```bash
docker compose up -d
```

`docker-compose.yml` 已包含 TURN 示例配置，可按需开启。

## 常用环境变量

| 变量 | 作用 |
|------|------|
| `HTTP_ADDR` | HTTP 监听地址，默认 `:8080`。 |
| `ALLOWED_ORIGIN` | CORS 白名单，生产建议填具体域名。 |
| `AUTH_TOKEN` / `ROOM_TOKENS` | 推流/拉流鉴权，支持房间级覆盖。 |
| `JWT_SECRET` | 启用 JWT 鉴权，`room` 字段限制房间，`role=admin` 访问管理接口。 |
| `RECORD_ENABLED` / `RECORD_DIR` | 控制录制与输出目录。 |
| `UPLOAD_RECORDINGS` 及 S3 相关变量 | 开启录制上传和对象存储参数。 |
| `MAX_SUBS_PER_ROOM` | 每个房间的订阅者上限。 |
| `RATE_LIMIT_RPS` / `RATE_LIMIT_BURST` | HTTP 接口限流阈值。 |
| `ADMIN_TOKEN` | 调用 `/api/admin/rooms/{room}/close` 的管理 Token。 |

更多变量可参考根目录 `README.md` 的完整表格。

## API 示例

```bash
# WHIP 推流（示意，通常由浏览器/OBS 完成）
curl -X POST "http://localhost:8080/api/whip/publish/demo" \
  -H "Content-Type: application/sdp" \
  --data-binary @offer.sdp

# 获取房间列表
curl http://localhost:8080/api/rooms | jq

# 关闭房间
curl -X POST \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  http://localhost:8080/api/admin/rooms/demo/close
```

## 常见问题

- **提示 `publisher already exists`**：同一房间默认仅允许一个发布者，请确保旧连接已关闭，或调用管理接口关闭房间。
- **浏览器无法建立连接**：检查浏览器是否使用 HTTPS 环境；若在公网/对称 NAT，需要配置 TURN 服务器并更新 `TURN_URLS`、`TURN_USERNAME`、`TURN_PASSWORD`。
- **录制文件缺失**：确认 `RECORD_ENABLED=1` 且进程对 `RECORD_DIR` 拥有写权限；若启用上传并打开 `DELETE_RECORDING_AFTER_UPLOAD=1`，文件会在成功上传后被删除，可通过对象存储验证。
- **限流误伤**：`RATE_LIMIT_RPS=0` 可关闭限流；调大 `RATE_LIMIT_BURST` 可容忍短时间抖动。

如需更深入的背景，可阅读 `docs/design.md`。
