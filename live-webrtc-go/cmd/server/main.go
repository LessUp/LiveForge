package main

import (
	"context"
	"embed"
	"io/fs"
	"fmt"
	"log"
	"os"
	"os/signal"
	"net/http"
	"strings"
	"syscall"
	"time"

	"live-webrtc-go/internal/config"
	"live-webrtc-go/internal/api"
	"live-webrtc-go/internal/sfu"
	"live-webrtc-go/internal/uploader"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

//go:embed web
var webFS embed.FS

func main() {
	cfg := config.Load()
	_ = uploader.Init(cfg)
	mgr := sfu.NewManager(cfg)
	h := api.NewHTTPHandlers(mgr, cfg)

	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/api/whip/publish/", func(w http.ResponseWriter, r *http.Request) {
		room := strings.TrimPrefix(r.URL.Path, "/api/whip/publish/")
		if room == "" || strings.Contains(room, "..") {
			http.Error(w, "invalid room", http.StatusBadRequest)
			return
		}
		h.ServeWHIPPublish(w, r, room)
	})

	mux.HandleFunc("/api/whep/play/", func(w http.ResponseWriter, r *http.Request) {
		room := strings.TrimPrefix(r.URL.Path, "/api/whep/play/")
		if room == "" || strings.Contains(room, "..") {
			http.Error(w, "invalid room", http.StatusBadRequest)
			return
		}
		h.ServeWHEPPlay(w, r, room)
	})

	// Rooms list
	mux.HandleFunc("/api/rooms", h.ServeRooms)
	mux.HandleFunc("/api/records", h.ServeRecordsList)

	// Admin close room: /api/admin/rooms/{room}/close
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

	// Health check
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.Handle("/metrics", promhttp.Handler())

	// Recorded files
	mux.Handle("/records/", http.StripPrefix("/records/", http.FileServer(http.Dir(cfg.RecordDir))))

	// Static files (embedded)
	staticFS, _ := fs.Sub(webFS, "web")
	mux.Handle("/web/", http.StripPrefix("/web/", http.FileServer(http.FS(staticFS))))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/web/index.html", http.StatusFound)
			return
		}
		http.NotFound(w, r)
	})

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

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	mgr.CloseAll()
}
