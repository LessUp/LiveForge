// 包 config 负责从环境变量加载运行时配置，给服务各模块使用。
package config

import (
	"os"
	"strconv"
	"strings"
)

// Config 汇总 HTTP 服务、SFU、录制、上传、鉴权等配置项。
type Config struct {
    HTTPAddr          string            // HTTP 服务监听地址，例如 ":8080"
    AllowedOrigin     string            // 允许的跨域来源，"*" 表示全部
    AuthToken         string            // 全局访问 Token（房间级优先）
    STUN              []string          // STUN 服务器 URL 列表
    TURN              []string          // TURN 服务器 URL 列表
    TLSCertFile       string            // TLS 证书文件路径（可选）
    TLSKeyFile        string            // TLS 私钥文件路径（可选）
    RecordEnabled     bool              // 是否开启录制
    RecordDir         string            // 录制文件存储目录
    MaxSubsPerRoom    int               // 每房间最大订阅者数（0 表示不限）
    RoomTokens        map[string]string // 房间级 Token 映射：room->token
    TURNUsername      string            // TURN 用户名
    TURNPassword      string            // TURN 密码
    UploadEnabled     bool              // 是否开启录制文件上传
    DeleteAfterUpload bool              // 上传成功后是否删除本地文件
    S3Endpoint        string            // 对象存储端点
    S3Region          string            // 对象存储区域（可选）
    S3Bucket          string            // 对象存储桶名
    S3AccessKey       string            // 访问密钥 ID
    S3SecretKey       string            // 访问密钥 Secret
    S3UseSSL          bool              // 是否使用 SSL 访问对象存储
    S3PathStyle       bool              // 是否使用 Path-Style 访问
    S3Prefix          string            // 上传时的对象名前缀
    AdminToken        string            // 管理接口的 Token
    RateLimitRPS      float64           // 每 IP 的速率限制（每秒请求数）
    RateLimitBurst    int               // 速率限制突发值
    JWTSecret         string            // JWT HMAC 密钥
    PprofEnabled      bool              // 是否启用 pprof 调试端点
}

// Load 会读取环境变量并填充 Config，使用合理的默认值。
// Load 从环境变量读取配置项并设置默认值，适合教学演示环境。
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
