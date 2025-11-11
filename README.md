# live-webrtc-go

使用 Go + [Pion WebRTC](https://github.com/pion/webrtc) 构建的轻量级在线直播服务示例。实现了 WHIP 推流、WHEP 播放、嵌入式 Web 页面、可配置鉴权与房间状态查询，可作为开源参考或项目脚手架。

## 功能特点

- **WebRTC SFU**：基于 Pion 实现最小可用的房间转发逻辑，支持多观众同时观看。
- **WHIP / WHEP 接口**：HTTP API 兼容现代浏览器或 OBS WHIP 插件推流与播放。
- **可选鉴权**：配置 `AUTH_TOKEN` 后，后端要求 `Authorization: Bearer <token>` 或 `X-Auth-Token` 请求头。
- **房间状态查询**：`GET /api/rooms` 返回在线房间、发布者与订阅者统计。
- **健康检查**：`GET /healthz`，便于部署活性探测。
- **内嵌前端**：简单的推流/播放页面，支持输入房间与 Token。
- **部署友好**：通过环境变量配置 CORS、STUN/TURN、TLS、订阅上限、按房间 Token 等。
- **录制能力**：可选将 VP8/VP9 保存为 IVF、Opus 保存为 OGG（开启 `RECORD_ENABLED=1`）。
- **监控指标**：`GET /metrics` 暴露 Prometheus 指标（RTP 字节/包、订阅者数、房间数）。
- **容器化**：提供 Dockerfile 与示例 docker-compose.yml，支持挂载录制目录。

## 快速开始

```bash
# 克隆仓库
 git clone https://github.com/<your-account>/live-webrtc-go.git
 cd live-webrtc-go

# （确保 go 已加入 PATH，推荐 Go 1.21+）
 go mod tidy
 go run ./cmd/server
```

### 启动脚本

本地开发可以直接使用 `scripts/start.sh`，脚本会自动准备缓存目录并可加载 `.env.local`：

```bash
./scripts/start.sh
```

设置 `SKIP_TIDY=1 ./scripts/start.sh` 可跳过 `go mod tidy`。

启动后访问：

- 推流页：http://localhost:8080/web/publisher.html
- 播放页：http://localhost:8080/web/player.html
- 房间列表：http://localhost:8080/api/rooms
- 健康检查：http://localhost:8080/healthz
- 指标监控：http://localhost:8080/metrics
- 录制访问：http://localhost:8080/records/

> 请允许浏览器使用麦克风/摄像头。若启用 `AUTH_TOKEN`，在页面的 Token 输入框填写相同值。

## HTTP API

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/api/whip/publish/{room}` | 接受 SDP Offer，返回 SDP Answer，建立推流连接 |
| `POST` | `/api/whep/play/{room}` | 接受 SDP Offer，返回 SDP Answer，建立播放连接 |
| `GET` | `/api/rooms` | 返回房间列表与在线状态 |
| `GET` | `/api/records` | 返回录制文件列表（名称/大小/时间/URL） |
| `POST` | `/api/admin/rooms/{room}/close` | 关闭指定房间（需 `ADMIN_TOKEN` 鉴权） |
| `GET` | `/healthz` | 健康检查 |

### 鉴权

若设置 `AUTH_TOKEN` 环境变量，以上需要鉴权的接口必须携带：

```
Authorization: Bearer <token>
```

或

```
X-Auth-Token: <token>
```

## 配置项（环境变量）

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `HTTP_ADDR` | `:8080` | HTTP 服务监听地址 |
| `ALLOWED_ORIGIN` | `*` | CORS 允许的 Origin，生产环境建议填写具体域名 |
| `AUTH_TOKEN` | _(空)_ | 全局 Token（可被房间级 Token 覆盖） |
| `ROOM_TOKENS` | _(空)_ | 房间级 Token，格式 `room1:tok1;room2:tok2` |
| `STUN_URLS` | `stun:stun.l.google.com:19302` | 逗号分隔的 STUN 服务器列表 |
| `TURN_URLS` | _(空)_ | 逗号分隔的 TURN 服务器列表（生产环境推荐配置） |
| `TURN_USERNAME` | _(空)_ | TURN 用户名（与 TURN_URLS 配合） |
| `TURN_PASSWORD` | _(空)_ | TURN 密码（与 TURN_URLS 配合） |
| `TLS_CERT_FILE` | _(空)_ | 启用 TLS 时的证书路径（配合 `TLS_KEY_FILE`） |
| `TLS_KEY_FILE` | _(空)_ | 启用 TLS 时的私钥路径 |
| `RECORD_ENABLED` | `0` | 设置为 `1` 启用录制功能 |
| `RECORD_DIR` | `records` | 录制文件保存目录（也用于 `/records/` 静态访问） |
| `MAX_SUBS_PER_ROOM` | `0` | 每房间订阅者上限，`0` 表示不限制 |
| `UPLOAD_RECORDINGS` | `0` | 设置为 `1` 启用录制文件上传 |
| `DELETE_RECORDING_AFTER_UPLOAD` | `0` | 设置为 `1` 上传成功后删除本地录制 |
| `S3_ENDPOINT` | _(空)_ | S3/MinIO 端点，如 `127.0.0.1:9000` 或 `s3.amazonaws.com` |
| `S3_REGION` | _(空)_ | 区域（AWS 需要），MinIO 可留空 |
| `S3_BUCKET` | _(空)_ | 目标桶名 |
| `S3_ACCESS_KEY` | _(空)_ | 访问 Key |
| `S3_SECRET_KEY` | _(空)_ | 访问 Secret |
| `S3_USE_SSL` | `1` | 是否使用 SSL（`1`/`0`） |
| `S3_PATH_STYLE` | `0` | 是否启用 Path-Style（MinIO 通常为 `1`） |
| `S3_PREFIX` | _(空)_ | 上传时的对象前缀，可为空 |
| `ADMIN_TOKEN` | _(空)_ | 管理员令牌，用于调用管理接口 |
| `RATE_LIMIT_RPS` | `0` | 每 IP 限流速率（请求/秒，`0` 表示关闭） |
| `RATE_LIMIT_BURST` | `0` | 限流突发容量（令牌桶大小） |

### 管理接口示例

```bash
curl -X POST \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  http://localhost:8080/api/admin/rooms/demo/close -i
```

### 关闭与优雅停机

服务收到中断信号（Ctrl+C 或 SIGTERM）后，将优雅关闭 HTTP 服务并关闭所有房间、连接与录制资源。

## 项目结构

```
├── cmd/server          # 入口程序（HTTP + WebRTC）
│   └── web              # 嵌入式静态页面
├── internal/api         # HTTP handlers (WHIP/WHEP/Rooms)
├── internal/config      # 配置加载
├── internal/metrics     # Prometheus 指标
├── internal/sfu         # WebRTC SFU 管理逻辑
├── go.mod / go.sum
├── .gitignore / .gitattributes
└── README.md
```

## 部署建议

1. **HTTPS / TLS**：浏览器 WebRTC 通常要求 HTTPS，生产部署可使用反向代理（Nginx/Caddy）或自签证书调试。
2. **TURN 服务**：网络环境受限时需要 TURN 中继，可搭配 coturn。
3. **容器化**：提供 `Dockerfile` 与 `docker-compose.yml`，支持挂载录制目录与配置环境变量。
4. **监控日志**：集成 Prometheus、OpenTelemetry 或外部日志系统以便运维。

## 容器化部署

构建镜像：

```bash
docker build -t live-webrtc-go:latest .
```

快速运行（挂载录制目录，开启录制与公开端口）：

```bash
docker run --rm -p 8080:8080 \
  -e RECORD_ENABLED=1 \
  -e RECORD_DIR=/records \
  -e STUN_URLS=stun:stun.l.google.com:19302 \
  -v "$PWD/records:/records" \
  live-webrtc-go:latest
```

使用 Compose：

```bash
docker compose up -d
```

## 后续开发路线

1. **鉴权与房间治理加强**：接入 JWT/OAuth，完善主播与观众角色管理、房间限流、黑名单等策略。
2. **录制与回放**：将推流内容落地为 MP4/TS，提供点播回放或上传至对象存储。
3. **转码与自适应码率**：集成 FFmpeg/GStreamer，实现多码率输出与网络自适应策略。
4. **监控与告警**：输出 Prometheus 指标（连麦数、比特率、丢包率、延迟估算），接入日志/追踪系统并配置告警。
5. **部署自动化**：提供 Dockerfile、Helm Chart 或 Terraform 模板，支持多环境交付与 CI/CD。
6. **网络与可靠性增强**：生产配置多节点 SFU、内置 TURN 服务，可选对接 CDN 或录制队列。
7. **前端体验完善**：构建独立 Web 客户端/后台管理界面，支持主持人控制、弹幕/聊天等互动功能。

## 许可协议

请根据你的开源计划选择合适的开源许可证，并将 `LICENSE` 文件加入仓库（例如 MIT、Apache-2.0 等）。

## 贡献指南

欢迎 Issue 与 Pull Request！

> 若你在实际项目中使用本仓库，欢迎分享反馈与改进建议。

## 延伸文档

- [使用指南](docs/usage.md)：完整的启动、部署、API 示例与排障。
- [设计说明](docs/design.md)：架构背景、模块拆分与数据流。
