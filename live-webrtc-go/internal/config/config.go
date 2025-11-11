package config

import (
	"os"
	"strconv"
	"strings"
)

// Config 汇总 HTTP 服务、SFU、录制、上传、鉴权等配置项。
type Config struct {
	HTTPAddr          string
	AllowedOrigin     string
	AuthToken         string
	STUN              []string
	TURN              []string
	TLSCertFile       string
	TLSKeyFile        string
	RecordEnabled     bool
	RecordDir         string
	MaxSubsPerRoom    int
	RoomTokens        map[string]string
	TURNUsername      string
	TURNPassword      string
	UploadEnabled     bool
	DeleteAfterUpload bool
	S3Endpoint        string
	S3Region          string
	S3Bucket          string
	S3AccessKey       string
	S3SecretKey       string
	S3UseSSL          bool
	S3PathStyle       bool
	S3Prefix          string
	AdminToken        string
	RateLimitRPS      float64
	RateLimitBurst    int
	JWTSecret         string
	PprofEnabled      bool
}

// Load 会读取环境变量并填充 Config，使用合理的默认值。
func Load() *Config {
	c := &Config{
		HTTPAddr:      getEnv("HTTP_ADDR", ":8080"),
		AllowedOrigin: getEnv("ALLOWED_ORIGIN", "*"),
		AuthToken:     getEnv("AUTH_TOKEN", ""),
	}
	if v := os.Getenv("STUN_URLS"); v != "" {
		c.STUN = splitCSV(v)
	} else {
		c.STUN = []string{"stun:stun.l.google.com:19302"}
	}
	if v := os.Getenv("TURN_URLS"); v != "" {
		c.TURN = splitCSV(v)
	}
	c.TURNUsername = getEnv("TURN_USERNAME", "")
	c.TURNPassword = getEnv("TURN_PASSWORD", "")
	c.TLSCertFile = getEnv("TLS_CERT_FILE", "")
	c.TLSKeyFile = getEnv("TLS_KEY_FILE", "")
	c.RecordEnabled = getEnv("RECORD_ENABLED", "") == "1"
	c.RecordDir = getEnv("RECORD_DIR", "records")
	if v := getEnv("MAX_SUBS_PER_ROOM", "0"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.MaxSubsPerRoom = n
		}
	}
	if v := os.Getenv("ROOM_TOKENS"); v != "" {
		c.RoomTokens = parseRoomTokens(v)
	} else {
		c.RoomTokens = map[string]string{}
	}
	c.UploadEnabled = getEnv("UPLOAD_RECORDINGS", "") == "1"
	c.DeleteAfterUpload = getEnv("DELETE_RECORDING_AFTER_UPLOAD", "") == "1"
	c.S3Endpoint = getEnv("S3_ENDPOINT", "")
	c.S3Region = getEnv("S3_REGION", "")
	c.S3Bucket = getEnv("S3_BUCKET", "")
	c.S3AccessKey = getEnv("S3_ACCESS_KEY", "")
	c.S3SecretKey = getEnv("S3_SECRET_KEY", "")
	c.S3UseSSL = getEnv("S3_USE_SSL", "1") == "1"
	c.S3PathStyle = getEnv("S3_PATH_STYLE", "") == "1"
	c.S3Prefix = getEnv("S3_PREFIX", "")
	c.AdminToken = getEnv("ADMIN_TOKEN", "")
	if v := getEnv("RATE_LIMIT_RPS", "0"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			c.RateLimitRPS = f
		}
	}
	if v := getEnv("RATE_LIMIT_BURST", "0"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.RateLimitBurst = n
		}
	}
	c.JWTSecret = getEnv("JWT_SECRET", "")
	c.PprofEnabled = getEnv("PPROF", "") == "1"
	return c
}

func getEnv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

// splitCSV 解析逗号分隔的列表，同时清理多余空白。
func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// parseRoomTokens 支持 "room1:token1;room2:token2" 风格的配置。
func parseRoomTokens(s string) map[string]string {
	m := map[string]string{}
	items := strings.Split(s, ";")
	for _, it := range items {
		it = strings.TrimSpace(it)
		if it == "" {
			continue
		}
		kv := strings.SplitN(it, ":", 2)
		if len(kv) != 2 {
			continue
		}
		k := strings.TrimSpace(kv[0])
		v := strings.TrimSpace(kv[1])
		if k != "" && v != "" {
			m[k] = v
		}
	}
	return m
}
