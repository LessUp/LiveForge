package main

import (
	"embed"
	"io/fs"
	"fmt"
	"log"
	"net/http"
	"strings"

	"live-webrtc-go/internal/config"
	"live-webrtc-go/internal/api"
	"live-webrtc-go/internal/sfu"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

//go:embed web
var webFS embed.FS

func main() {
	cfg := config.Load()
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
	if cfg.TLSCertFile != "" && cfg.TLSKeyFile != "" {
		if err := http.ListenAndServeTLS(addr, cfg.TLSCertFile, cfg.TLSKeyFile, mux); err != nil {
			log.Fatal(err)
		}
		return
	}
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
