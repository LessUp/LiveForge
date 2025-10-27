package api

import (
    "encoding/json"
    "io"
    "os"
    "path/filepath"
    "net"
    "net/http"
    "strings"
    "time"

    "live-webrtc-go/internal/config"
    "live-webrtc-go/internal/sfu"
)

type HTTPHandlers struct {
    mgr *sfu.Manager
    cfg *config.Config
}

// ServeRooms handles GET /api/rooms
func (h *HTTPHandlers) ServeRooms(w http.ResponseWriter, r *http.Request) {
    h.allowCORS(w, r)
    if r.Method == http.MethodOptions {
        w.WriteHeader(http.StatusNoContent)
        return
    }
    if r.Method != http.MethodGet {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }
    rooms := h.mgr.ListRooms()
    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(rooms)
}

func NewHTTPHandlers(m *sfu.Manager, c *config.Config) *HTTPHandlers {
    return &HTTPHandlers{mgr: m, cfg: c}
}

// ServeWHIPPublish handles POST /api/whip/publish/{room}
func (h *HTTPHandlers) ServeWHIPPublish(w http.ResponseWriter, r *http.Request, room string) {
    h.allowCORS(w, r)
    if r.Method == http.MethodOptions {
        w.WriteHeader(http.StatusNoContent)
        return
    }
    if r.Method != http.MethodPost {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }
    if !h.authOKRoom(r, room) {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }
    defer r.Body.Close()
    offerSDP, _ := io.ReadAll(r.Body)
    answer, err := h.mgr.Publish(r.Context(), room, string(offerSDP))
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    w.Header().Set("Content-Type", "application/sdp")
    w.WriteHeader(http.StatusCreated)
    _, _ = w.Write([]byte(answer))
}

// ServeWHEPPlay handles POST /api/whep/play/{room}
func (h *HTTPHandlers) ServeWHEPPlay(w http.ResponseWriter, r *http.Request, room string) {
    h.allowCORS(w, r)
    if r.Method == http.MethodOptions {
        w.WriteHeader(http.StatusNoContent)
        return
    }
    if r.Method != http.MethodPost {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }
    if !h.authOKRoom(r, room) {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }
    defer r.Body.Close()
    offerSDP, _ := io.ReadAll(r.Body)
    answer, err := h.mgr.Subscribe(r.Context(), room, string(offerSDP))
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    w.Header().Set("Content-Type", "application/sdp")
    w.WriteHeader(http.StatusCreated)
    _, _ = w.Write([]byte(answer))
}

func (h *HTTPHandlers) allowCORS(w http.ResponseWriter, r *http.Request) {
    origin := r.Header.Get("Origin")
    ao := h.cfg.AllowedOrigin
    if ao == "*" {
        w.Header().Set("Access-Control-Allow-Origin", "*")
    } else if origin != "" && (ao == origin || hostMatch(ao, origin)) {
        w.Header().Set("Access-Control-Allow-Origin", origin)
        w.Header().Set("Vary", "Origin")
    }
    w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
    w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Auth-Token")
    w.Header().Set("Access-Control-Allow-Credentials", "true")
}

func (h *HTTPHandlers) authOKRoom(r *http.Request, room string) bool {
    // room-specific token overrides global config if set
    if tok, ok := h.cfg.RoomTokens[room]; ok && tok != "" {
        return tokenMatch(r, tok)
    }
    if h.cfg.AuthToken == "" {
        return true
    }
    return tokenMatch(r, h.cfg.AuthToken)
}

func tokenMatch(r *http.Request, expect string) bool {
    if t := r.Header.Get("X-Auth-Token"); t != "" {
        return t == expect
    }
    auth := r.Header.Get("Authorization")
    if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
        return strings.TrimSpace(auth[7:]) == expect
    }
    return false
}

func hostMatch(expect, origin string) bool {
    u := origin
    if i := strings.Index(origin, "://"); i >= 0 {
        u = origin[i+3:]
    }
    if j := strings.Index(u, "/"); j >= 0 {
        u = u[:j]
    }
    host, _, err := net.SplitHostPort(u)
    if err != nil {
        host = u
    }
    return host == expect || origin == expect
}

func (h *HTTPHandlers) ServeRecordsList(w http.ResponseWriter, r *http.Request) {
    h.allowCORS(w, r)
    if r.Method == http.MethodOptions {
        w.WriteHeader(http.StatusNoContent)
        return
    }
    if r.Method != http.MethodGet {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }
    dir := h.cfg.RecordDir
    entries, err := os.ReadDir(dir)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    type rec struct {
        Name    string `json:"name"`
        Size    int64  `json:"size"`
        ModTime string `json:"modTime"`
        URL     string `json:"url"`
    }
    var list []rec
    for _, e := range entries {
        if e.IsDir() { continue }
        name := e.Name()
        ext := strings.ToLower(filepath.Ext(name))
        if ext != ".ivf" && ext != ".ogg" { continue }
        fi, err := e.Info()
        if err != nil { continue }
        list = append(list, rec{
            Name: name,
            Size: fi.Size(),
            ModTime: fi.ModTime().UTC().Format(time.RFC3339),
            URL: "/records/" + name,
        })
    }
    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(list)
}
