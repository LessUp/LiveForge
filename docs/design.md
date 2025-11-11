# 设计说明

本项目实现了一个轻量级 WebRTC SFU，核心目标是提供 **WHIP 推流 + WHEP 播放** 的最小可用能力，并在 Go 语言栈内集成鉴权、限流、录制、上传与指标观测。下文描述总体架构、模块拆分和数据流向，便于二次开发或评审。

## 系统概览

- **协议**：后端提供 RESTful API，分别处理 `/api/whip/publish/{room}` 与 `/api/whep/play/{room}`，配合嵌入式静态页面即可推/拉一间房间。
- **SFU**：基于 Pion WebRTC，单发布者、多订阅者结构，集中做 Track fanout，观众之间不会互相连线。
- **部署模型**：一个进程同时承载 HTTP 服务、Web 静态资源、录制目录与可选的对象存储上传能力，可通过 Docker/K8s 扩缩。
- **监控治理**：Prometheus 指标追踪房间/订阅者/RTP 数据量，`/api/rooms` 可视化实时状态，`ADMIN_TOKEN`、JWT、限流器负责访问治理。

```
┌─────────────┐    Publish (WHIP)    ┌──────────────┐
│  Publisher  │ ───────────────────▶ │ HTTP Handlers│
└─────────────┘                      └──────┬───────┘
                                            │
                                SFU Fanout  │
                                            ▼
                                       ┌────────┐
┌─────────────┐   Play (WHEP)    ◀──── │  Room  │
│   Player    │──────────────────────▶ │        │
└─────────────┘                       └────────┘
                                            │
                                 Record/Upload│
                                            ▼
                                       Object Storage
```

## 模块拆分

| 模块 | 说明 |
|------|------|
| `cmd/server` | 入口程序：加载配置、注册 HTTP 路由、内嵌静态页面、托管 Prometheus 指标与优雅退出逻辑。 |
| `internal/config` | 从环境变量解析配置，提供 STUN/TURN、鉴权、录制、上传、限流等默认值。 |
| `internal/api` | 实现 WHIP/WHEP/房间列表/录制列表/管理接口，含 CORS、鉴权、速率限制等横切逻辑。 |
| `internal/sfu` | 房间及 track fanout 的核心实现：管理 PeerConnection 生命周期、录制落盘并统计指标。 |
| `internal/metrics` | Prometheus 指标定义，追踪房间数、订阅者数以及 RTP 字节/包。 |
| `internal/uploader` | 可选的录制文件上传 S3/MinIO，支持上传后删除本地文件。 |
| `web` & `cmd/server/web` | 提供推流/播放/房间列表等示例页面，方便快速体验。 |

## 数据流与关键逻辑

### 推流（WHIP）
1. 前端或 OBS 通过 HTTP `POST /api/whip/publish/{room}` 发送 SDP Offer。
2. `HTTPHandlers` 校验 Token/JWT，并触发 `Manager.Publish`。
3. `Room.Publish` 创建专属 PeerConnection，只允许单个发布者；为每条 Track 创建 `trackFanout` 和可选的录制写入。
4. SFU 稳定后返回 SDP Answer，客户端进入推流状态；若 enable 录制，则持续写入 IVF/OGG，并异步上传。

### 播放（WHEP）
1. 观众发送 `POST /api/whep/play/{room}`。
2. 通过鉴权与限流后，`Room.Subscribe` 检查订阅上限并创建新的 PeerConnection。
3. 每个现有 trackFanout 会 `attachToSubscriber` 创建本地 Track，随后订阅者就绪。
4. 当订阅者断开或出现 ICE Failure 时，`removeSubscriber` 会清理资源并更新指标。

### 录制与上传
- 录制由 `trackFanout` 触发：检测到 Opus/VP8/VP9 即写入 OGG/IVF，文件存储于 `RECORD_DIR`。
- 关闭房间或 track 时会关闭写入器，并调用 `uploader.Upload` 在后台将文件推送到对象存储（若已启用）。
- `ServeRecordsList` 读取目录返回元数据，可配合 `/records/` 静态服务或外部下载。

## 配置与扩展点

- **鉴权**：支持三层优先级——房间级 Token、全局 Token、JWT。JWT 可通过 `room` 声明限制访问，也可通过 `role/admin` 声明访问管理接口。
- **限流**：`RATE_LIMIT_RPS` + `RATE_LIMIT_BURST` 基于 IP 的简单令牌桶，防止接口被滥用。
- **网络**：支持自定义 STUN/TURN，若提供 TLS 证书可直接 `ListenAndServeTLS`。
- **可观测性**：`/metrics` 提供 Prometheus 指标，`/healthz` 用于探活。
- **水平扩展**：房间信息在内存中维护，适合单实例或借助外部编排做会话亲和；若需多实例，可引入 Redis/数据库做房间状态共享。

## 运行与调试建议

- 推荐使用 `scripts/start.sh` 本地启动，统一处理 `GOCACHE`/`GOMODCACHE` 并可加载 `.env.local`。
- 生产环境建议放置在反向代理之后，开启 HTTPS、配置 TURN，并将录制目录挂载到持久化存储。
- 对性能敏感的场景，可针对 `trackFanout` 做零拷贝优化，或改用真正的 SFU 框架（如 ion-sfu、livekit）。

## 后续演进方向

1. 多房间分布式管理：将房间元数据与订阅状态存入 Redis，支持节点故障转移。
2. 媒体处理链路：接入 FFmpeg 进行转码、多码率或截图。
3. 运维治理：接入 OpenTelemetry，记录关键事件与耗时，结合指标实现自动扩容/告警。
4. 安全强化：补充鉴权回调、IP 白名单、推流限时与房间禁播策略。
