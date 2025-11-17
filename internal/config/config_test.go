package config

import (
	"os"
	"testing"
)

func TestLoad_DefaultValues(t *testing.T) {
	// Clean environment
	os.Clearenv()
	
	cfg := Load()
	
	// Test default values
	if cfg.HTTPAddr != ":8080" {
		t.Errorf("Expected HTTPAddr to be :8080, got %s", cfg.HTTPAddr)
	}
	
	if cfg.AllowedOrigin != "*" {
		t.Errorf("Expected AllowedOrigin to be *, got %s", cfg.AllowedOrigin)
	}
	
	if cfg.RecordDir != "records" {
		t.Errorf("Expected RecordDir to be records, got %s", cfg.RecordDir)
	}
	
	if cfg.MaxSubsPerRoom != 0 {
		t.Errorf("Expected MaxSubsPerRoom to be 0, got %d", cfg.MaxSubsPerRoom)
	}
	
	if cfg.RateLimitRPS != 0 {
		t.Errorf("Expected RateLimitRPS to be 0, got %f", cfg.RateLimitRPS)
	}
	
	if len(cfg.STUN) != 1 || cfg.STUN[0] != "stun:stun.l.google.com:19302" {
		t.Errorf("Expected default STUN server, got %v", cfg.STUN)
	}
}

func TestLoad_EnvironmentVariables(t *testing.T) {
	// Set test environment variables
	envVars := map[string]string{
		"HTTP_ADDR":          ":9090",
		"ALLOWED_ORIGIN":     "https://example.com",
		"AUTH_TOKEN":         "test-token",
		"STUN_URLS":          "stun:stun1.example.com:3478,stun:stun2.example.com:3478",
		"TURN_URLS":          "turn:turn.example.com:3478",
		"TURN_USERNAME":      "testuser",
		"TURN_PASSWORD":      "testpass",
		"TLS_CERT_FILE":      "/path/to/cert.pem",
		"TLS_KEY_FILE":       "/path/to/key.pem",
		"RECORD_ENABLED":     "1",
		"RECORD_DIR":         "/custom/records",
		"MAX_SUBS_PER_ROOM":  "50",
		"ROOM_TOKENS":        "room1:token1;room2:token2",
		"UPLOAD_RECORDINGS": "1",
		"DELETE_RECORDING_AFTER_UPLOAD": "1",
		"S3_ENDPOINT":        "s3.amazonaws.com",
		"S3_REGION":          "us-east-1",
		"S3_BUCKET":          "test-bucket",
		"S3_ACCESS_KEY":      "access-key",
		"S3_SECRET_KEY":      "secret-key",
		"S3_USE_SSL":         "1",
		"S3_PATH_STYLE":      "0",
		"S3_PREFIX":          "recordings/",
		"ADMIN_TOKEN":        "admin-token",
		"RATE_LIMIT_RPS":     "10.5",
		"RATE_LIMIT_BURST":   "20",
		"JWT_SECRET":         "jwt-secret",
		"PPROF":              "1",
	}
	
	// Set environment variables
	for k, v := range envVars {
		os.Setenv(k, v)
	}
	defer func() {
		// Clean up
		for k := range envVars {
			os.Unsetenv(k)
		}
	}()
	
	cfg := Load()
	
	// Test loaded values
	if cfg.HTTPAddr != ":9090" {
		t.Errorf("Expected HTTPAddr to be :9090, got %s", cfg.HTTPAddr)
	}
	
	if cfg.AllowedOrigin != "https://example.com" {
		t.Errorf("Expected AllowedOrigin to be https://example.com, got %s", cfg.AllowedOrigin)
	}
	
	if cfg.AuthToken != "test-token" {
		t.Errorf("Expected AuthToken to be test-token, got %s", cfg.AuthToken)
	}
	
	if len(cfg.STUN) != 2 {
		t.Errorf("Expected 2 STUN servers, got %d", len(cfg.STUN))
	}
	
	if len(cfg.TURN) != 1 {
		t.Errorf("Expected 1 TURN server, got %d", len(cfg.TURN))
	}
	
	if cfg.TURNUsername != "testuser" {
		t.Errorf("Expected TURNUsername to be testuser, got %s", cfg.TURNUsername)
	}
	
	if cfg.TURNPassword != "testpass" {
		t.Errorf("Expected TURNPassword to be testpass, got %s", cfg.TURNPassword)
	}
	
	if !cfg.RecordEnabled {
		t.Error("Expected RecordEnabled to be true")
	}
	
	if cfg.RecordDir != "/custom/records" {
		t.Errorf("Expected RecordDir to be /custom/records, got %s", cfg.RecordDir)
	}
	
	if cfg.MaxSubsPerRoom != 50 {
		t.Errorf("Expected MaxSubsPerRoom to be 50, got %d", cfg.MaxSubsPerRoom)
	}
	
	if len(cfg.RoomTokens) != 2 {
		t.Errorf("Expected 2 room tokens, got %d", len(cfg.RoomTokens))
	}
	
	if cfg.RoomTokens["room1"] != "token1" {
		t.Errorf("Expected room1 token to be token1, got %s", cfg.RoomTokens["room1"])
	}
	
	if !cfg.UploadEnabled {
		t.Error("Expected UploadEnabled to be true")
	}
	
	if !cfg.DeleteAfterUpload {
		t.Error("Expected DeleteAfterUpload to be true")
	}
	
	if cfg.S3Endpoint != "s3.amazonaws.com" {
		t.Errorf("Expected S3Endpoint to be s3.amazonaws.com, got %s", cfg.S3Endpoint)
	}
	
	if cfg.S3Region != "us-east-1" {
		t.Errorf("Expected S3Region to be us-east-1, got %s", cfg.S3Region)
	}
	
	if cfg.S3Bucket != "test-bucket" {
		t.Errorf("Expected S3Bucket to be test-bucket, got %s", cfg.S3Bucket)
	}
	
	if cfg.S3AccessKey != "access-key" {
		t.Errorf("Expected S3AccessKey to be access-key, got %s", cfg.S3AccessKey)
	}
	
	if cfg.S3SecretKey != "secret-key" {
		t.Errorf("Expected S3SecretKey to be secret-key, got %s", cfg.S3SecretKey)
	}
	
	if !cfg.S3UseSSL {
		t.Error("Expected S3UseSSL to be true")
	}
	
	if cfg.S3PathStyle {
		t.Error("Expected S3PathStyle to be false")
	}
	
	if cfg.S3Prefix != "recordings/" {
		t.Errorf("Expected S3Prefix to be recordings/, got %s", cfg.S3Prefix)
	}
	
	if cfg.AdminToken != "admin-token" {
		t.Errorf("Expected AdminToken to be admin-token, got %s", cfg.AdminToken)
	}
	
	if cfg.RateLimitRPS != 10.5 {
		t.Errorf("Expected RateLimitRPS to be 10.5, got %f", cfg.RateLimitRPS)
	}
	
	if cfg.RateLimitBurst != 20 {
		t.Errorf("Expected RateLimitBurst to be 20, got %d", cfg.RateLimitBurst)
	}
	
	if cfg.JWTSecret != "jwt-secret" {
		t.Errorf("Expected JWTSecret to be jwt-secret, got %s", cfg.JWTSecret)
	}
	
	if !cfg.PprofEnabled {
		t.Error("Expected PprofEnabled to be true")
	}
}

func TestSplitCSV(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{
			input:    "stun:stun1.example.com:3478,stun:stun2.example.com:3478",
			expected: []string{"stun:stun1.example.com:3478", "stun:stun2.example.com:3478"},
		},
		{
			input:    "stun:stun1.example.com:3478, stun:stun2.example.com:3478",
			expected: []string{"stun:stun1.example.com:3478", "stun:stun2.example.com:3478"},
		},
		{
			input:    "stun:stun1.example.com:3478",
			expected: []string{"stun:stun1.example.com:3478"},
		},
		{
			input:    "",
			expected: []string{},
		},
	}
	
	for _, test := range tests {
		result := splitCSV(test.input)
		if len(result) != len(test.expected) {
			t.Errorf("For input '%s', expected %d items, got %d", test.input, len(test.expected), len(result))
			continue
		}
		for i, expected := range test.expected {
			if result[i] != expected {
				t.Errorf("For input '%s', expected item %d to be '%s', got '%s'", test.input, i, expected, result[i])
			}
		}
	}
}

func TestParseRoomTokens(t *testing.T) {
	tests := []struct {
		input    string
		expected map[string]string
	}{
		{
			input: "room1:token1;room2:token2",
			expected: map[string]string{
				"room1": "token1",
				"room2": "token2",
			},
		},
		{
			input: "room1:token1; room2:token2",
			expected: map[string]string{
				"room1": "token1",
				"room2": "token2",
			},
		},
		{
			input: "room1:token1",
			expected: map[string]string{
				"room1": "token1",
			},
		},
		{
			input:    "",
			expected: map[string]string{},
		},
		{
			input: "invalid_format",
			expected: map[string]string{},
		},
	}
	
	for _, test := range tests {
		result := parseRoomTokens(test.input)
		if len(result) != len(test.expected) {
			t.Errorf("For input '%s', expected %d tokens, got %d", test.input, len(test.expected), len(result))
			continue
		}
		for k, expectedV := range test.expected {
			if result[k] != expectedV {
				t.Errorf("For input '%s', expected token for room '%s' to be '%s', got '%s'", test.input, k, expectedV, result[k])
			}
		}
	}
}

func TestGetEnv(t *testing.T) {
	// Test with existing environment variable
	os.Setenv("TEST_VAR", "test_value")
	defer os.Unsetenv("TEST_VAR")
	
	result := getEnv("TEST_VAR", "default")
	if result != "test_value" {
		t.Errorf("Expected getEnv to return 'test_value', got '%s'", result)
	}
	
	// Test with non-existing environment variable
	result = getEnv("NON_EXISTING_VAR", "default")
	if result != "default" {
		t.Errorf("Expected getEnv to return 'default', got '%s'", result)
	}
}