package config

import (
	"os"
	"strings"
)

type Config struct {
	HTTPAddr      string
	AllowedOrigin string
	AuthToken     string
	STUN          []string
	TURN          []string
}

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
	return c
}

func getEnv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

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
