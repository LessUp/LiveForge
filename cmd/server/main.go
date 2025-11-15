// 程序入口：启动一个轻量级 WebRTC 服务，提供 WHIP 推流与 WHEP 播放接口，
// 同时暴露房间/录制查询、Prometheus 指标与健康检查，并内嵌示例网页。
package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"live-webrtc-go/internal/api"
	"live-webrtc-go/internal/config"
	"live-webrtc-go/internal/sfu"
	"live-webrtc-go/internal/uploader"
)

// web 目录下的静态资源打包进二进制，便于教学演示与单文件部署。
//go:embed web
var webFS embed.FS

// main 负责：
// 1) 加载配置并初始化上传器与房间管理器
// 2) 注册 HTTP 路由（WHIP/WHEP/房间/录制/管理/指标/健康检查/静态页面）
// 3) 启动 HTTP/HTTPS 服务并实现优雅退出
func main() {
	// 加载配置并初始化依赖（上传器、SFU 管理器、HTTP 处理器）
	cfg := config.Load()
	_ = uploader.Init(cfg)
	mgr := sfu.NewManager(cfg)
	h := api.NewHTTPHandlers(mgr, cfg)

    // 使用标准库 ServeMux 注册各类路由
    mux := http.NewServeMux()

    // API：WHIP 推流（POST）
    mux.HandleFunc("/api/whip/publish/", func(w http.ResponseWriter, r *http.Request) {
        room := strings.TrimPrefix(r.URL.Path, "/api/whip/publish/")
        if room == "" || strings.Contains(room, "..") {
            http.Error(w, "invalid room", http.StatusBadRequest)
            return
        }
        h.ServeWHIPPublish(w, r, room)
    })

    // API：WHEP 播放（POST）
    mux.HandleFunc("/api/whep/play/", func(w http.ResponseWriter, r *http.Request) {
        room := strings.TrimPrefix(r.URL.Path, "/api/whep/play/")
        if room == "" || strings.Contains(room, "..") {
            http.Error(w, "invalid room", http.StatusBadRequest)
            return
        }
        h.ServeWHEPPlay(w, r, room)
    })

    // API：房间列表与录制文件列表（GET）
    mux.HandleFunc("/api/rooms", h.ServeRooms)
    mux.HandleFunc("/api/records", h.ServeRecordsList)

    // 管理接口：关闭房间（POST /api/admin/rooms/{room}/close）
    mux.HandleFunc("/api/admin/rooms/", func(w http.ResponseWriter, r *http.Request) {
        p := strings.TrimPrefix(r.URL.Path, "/api/admin/rooms/")
        if strings.HasSuffix(p, "/close") {
            room := strings.TrimSuffix(p, "/close")
            room = strings.TrimSuffix(room, "/")
            if room == "" || strings.Contains(room, "..") {
                http.Error(w, "invalid room", http.StatusBadRequest)
                return
            }
            h.ServeAdminCloseRoom(w, r, room)
            return
        }
        http.NotFound(w, r)
    })

    // 健康检查：用于存活探测与基础监控
    mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte("ok"))
    })

    // Prometheus 指标：采集房间数量、订阅者数、RTP 字节/包等
    mux.Handle("/metrics", promhttp.Handler())

    // 录制文件静态服务：直接暴露 RECORD_DIR 下内容
    mux.Handle("/records/", http.StripPrefix("/records/", http.FileServer(http.Dir(cfg.RecordDir))))

    // 内嵌静态页面：publisher.html / player.html 等示例
    staticFS, _ := fs.Sub(webFS, "web")
    mux.Handle("/web/", http.StripPrefix("/web/", http.FileServer(http.FS(staticFS))))
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path == "/" {
            http.Redirect(w, r, "/web/index.html", http.StatusFound)
            return
        }
        http.NotFound(w, r)
    })

    // 启动服务：根据是否配置证书选择 HTTP 或 HTTPS
    addr := cfg.HTTPAddr
    fmt.Printf("Live WebRTC server listening on %s\n", addr)
    fmt.Println("Open http://localhost:8080/web/publisher.html and http://localhost:8080/web/player.html")

    srv := &http.Server{Addr: addr, Handler: mux}
    go func() {
        var err error
        if cfg.TLSCertFile != "" && cfg.TLSKeyFile != "" {
            err = srv.ListenAndServeTLS(cfg.TLSCertFile, cfg.TLSKeyFile)
        } else {
            err = srv.ListenAndServe()
        }
        if err != nil && err != http.ErrServerClosed {
            log.Fatal(err)
        }
    }()

    // 优雅退出：捕获中断信号，优雅关闭 HTTP 并清理房间连接
    stop := make(chan os.Signal, 1)
    signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
    <-stop
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    _ = srv.Shutdown(ctx)
    mgr.CloseAll()
}
