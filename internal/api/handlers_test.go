package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"live-webrtc-go/internal/config"
	"live-webrtc-go/internal/sfu"
)

func setupTestHandlers() (*HTTPHandlers, *config.Config) {
	cfg := &config.Config{
		HTTPAddr:          ":8080",
		AllowedOrigin:     "*",
		AuthToken:         "",
		STUN:              []string{"stun:stun.l.google.com:19302"},
		TURN:              []string{},
		TLSCertFile:       "",
		TLSKeyFile:        "",
		RecordEnabled:     false,
		RecordDir:         "records",
		MaxSubsPerRoom:    0,
		RoomTokens:        map[string]string{},
		TURNUsername:      "",
		TURNPassword:      "",
		UploadEnabled:     false,
		DeleteAfterUpload: false,
		S3Endpoint:        "",
		S3Region:          "",
		S3Bucket:          "",
		S3AccessKey:       "",
		S3SecretKey:       "",
		S3UseSSL:          true,
		S3PathStyle:       false,
		S3Prefix:          "",
		AdminToken:        "",
		RateLimitRPS:      0,
		RateLimitBurst:    0,
		JWTSecret:         "",
		PprofEnabled:      false,
	}
	
	mgr := sfu.NewManager(cfg)
	h := NewHTTPHandlers(mgr, cfg)
	
	return h, cfg
}

func TestServeRooms_Success(t *testing.T) {
	h, _ := setupTestHandlers()
	
	req := httptest.NewRequest("GET", "/api/rooms", nil)
	w := httptest.NewRecorder()
	
	h.ServeRooms(w, req)
	
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	
	var rooms []sfu.RoomInfo
	err := json.NewDecoder(resp.Body).Decode(&rooms)
	if err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}
	
	if rooms == nil {
		t.Error("Expected empty rooms array, got nil")
	}
}

func TestServeRooms_OptionsMethod(t *testing.T) {
	h, _ := setupTestHandlers()
	
	req := httptest.NewRequest("OPTIONS", "/api/rooms", nil)
	w := httptest.NewRecorder()
	
	h.ServeRooms(w, req)
	
	resp := w.Result()
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", resp.StatusCode)
	}
}

func TestServeRooms_InvalidMethod(t *testing.T) {
	h, _ := setupTestHandlers()
	
	req := httptest.NewRequest("POST", "/api/rooms", nil)
	w := httptest.NewRecorder()
	
	h.ServeRooms(w, req)
	
	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

func TestServeWHIPPublish_Success(t *testing.T) {
	h, cfg := setupTestHandlers()
	
	// Add a test room token for authentication
	cfg.RoomTokens["test-room"] = "test-token"
	
	sdpOffer := "v=0\r\no=- 1234567890 1234567890 IN IP4 127.0.0.1\r\ns=-\r\nt=0 0\r\n"
	
	req := httptest.NewRequest("POST", "/api/whip/publish/test-room", strings.NewReader(sdpOffer))
	req.Header.Set("X-Auth-Token", "test-token")
	req.Header.Set("Content-Type", "application/sdp")
	w := httptest.NewRecorder()
	
	h.ServeWHIPPublish(w, req, "test-room")
	
	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		// We expect bad request because we don't have a valid WebRTC offer
		// but we want to test the handler flow
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status 400 (bad SDP), got %d", resp.StatusCode)
		}
	}
}

func TestServeWHIPPublish_NoAuth(t *testing.T) {
	h, cfg := setupTestHandlers()
	
	// Set auth token requirement
	cfg.AuthToken = "required-token"
	
	sdpOffer := "v=0\r\no=- 1234567890 1234567890 IN IP4 127.0.0.1\r\ns=-\r\nt=0 0\r\n"
	
	req := httptest.NewRequest("POST", "/api/whip/publish/test-room", strings.NewReader(sdpOffer))
	// No auth header
	w := httptest.NewRecorder()
	
	h.ServeWHIPPublish(w, req, "test-room")
	
	resp := w.Result()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestServeWHIPPublish_InvalidMethod(t *testing.T) {
	h, _ := setupTestHandlers()
	
	req := httptest.NewRequest("GET", "/api/whip/publish/test-room", nil)
	w := httptest.NewRecorder()
	
	h.ServeWHIPPublish(w, req, "test-room")
	
	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

func TestServeWHEPPlay_Success(t *testing.T) {
	h, cfg := setupTestHandlers()
	
	// Add a test room token for authentication
	cfg.RoomTokens["test-room"] = "test-token"
	
	sdpOffer := "v=0\r\no=- 1234567890 1234567890 IN IP4 127.0.0.1\r\ns=-\r\nt=0 0\r\n"
	
	req := httptest.NewRequest("POST", "/api/whep/play/test-room", strings.NewReader(sdpOffer))
	req.Header.Set("X-Auth-Token", "test-token")
	req.Header.Set("Content-Type", "application/sdp")
	w := httptest.NewRecorder()
	
	h.ServeWHEPPlay(w, req, "test-room")
	
	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		// We expect bad request because we don't have a valid WebRTC offer
		// but we want to test the handler flow
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status 400 (bad SDP), got %d", resp.StatusCode)
		}
	}
}

func TestServeWHEPPlay_NoAuth(t *testing.T) {
	h, cfg := setupTestHandlers()
	
	// Set auth token requirement
	cfg.AuthToken = "required-token"
	
	sdpOffer := "v=0\r\no=- 1234567890 1234567890 IN IP4 127.0.0.1\r\ns=-\r\nt=0 0\r\n"
	
	req := httptest.NewRequest("POST", "/api/whep/play/test-room", strings.NewReader(sdpOffer))
	// No auth header
	w := httptest.NewRecorder()
	
	h.ServeWHEPPlay(w, req, "test-room")
	
	resp := w.Result()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestServeRecordsList_Success(t *testing.T) {
	h, cfg := setupTestHandlers()
	
	// Create a temporary directory for records
	tempDir := t.TempDir()
	cfg.RecordDir = tempDir
	
	// Create a test recording file
	testFile := "test.ivf"
	testContent := []byte("test ivf content")
	err := os.WriteFile(tempDir+"/"+testFile, testContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	req := httptest.NewRequest("GET", "/api/records", nil)
	w := httptest.NewRecorder()
	
	h.ServeRecordsList(w, req)
	
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	
	var records []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&records)
	if err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}
	
	if len(records) != 1 {
		t.Errorf("Expected 1 record, got %d", len(records))
	}
	
	if records[0]["name"] != testFile {
		t.Errorf("Expected record name to be %s, got %v", testFile, records[0]["name"])
	}
}

func TestServeRecordsList_InvalidMethod(t *testing.T) {
	h, _ := setupTestHandlers()
	
	req := httptest.NewRequest("POST", "/api/records", nil)
	w := httptest.NewRecorder()
	
	h.ServeRecordsList(w, req)
	
	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

func TestServeAdminCloseRoom_Success(t *testing.T) {
	h, cfg := setupTestHandlers()
	
	// Set admin token
	cfg.AdminToken = "admin-token"
	
	// Create a room first
	mgr := sfu.NewManager(cfg)
	mgr.Publish(nil, "test-room", "invalid-sdp")
	
	req := httptest.NewRequest("POST", "/api/admin/rooms/test-room/close", nil)
	req.Header.Set("Authorization", "Bearer admin-token")
	w := httptest.NewRecorder()
	
	h.ServeAdminCloseRoom(w, req, "test-room")
	
	resp := w.Result()
	if resp.StatusCode != http.StatusNotFound {
		// Room doesn't exist, so we expect not found
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", resp.StatusCode)
		}
	}
}

func TestServeAdminCloseRoom_NoAuth(t *testing.T) {
	h, cfg := setupTestHandlers()
	
	// Set admin token
	cfg.AdminToken = "admin-token"
	
	req := httptest.NewRequest("POST", "/api/admin/rooms/test-room/close", nil)
	// No auth header
	w := httptest.NewRecorder()
	
	h.ServeAdminCloseRoom(w, req, "test-room")
	
	resp := w.Result()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestTokenMatch(t *testing.T) {
	tests := []struct {
		name     string
		header   map[string]string
		expected string
		result   bool
	}{
		{
			name:     "X-Auth-Token match",
			header:   map[string]string{"X-Auth-Token": "test-token"},
			expected: "test-token",
			result:   true,
		},
		{
			name:     "Authorization Bearer match",
			header:   map[string]string{"Authorization": "Bearer test-token"},
			expected: "test-token",
			result:   true,
		},
		{
			name:     "Authorization Bearer case insensitive",
			header:   map[string]string{"Authorization": "bearer test-token"},
			expected: "test-token",
			result:   true,
		},
		{
			name:     "No match",
			header:   map[string]string{"X-Auth-Token": "wrong-token"},
			expected: "test-token",
			result:   false,
		},
		{
			name:     "No auth header",
			header:   map[string]string{},
			expected: "test-token",
			result:   false,
		},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			for k, v := range test.header {
				req.Header.Set(k, v)
			}
			
			result := tokenMatch(req, test.expected)
			if result != test.result {
				t.Errorf("Expected tokenMatch to return %v, got %v", test.result, result)
			}
		})
	}
}

func TestAllowCORS(t *testing.T) {
	h, cfg := setupTestHandlers()
	
	tests := []struct {
		name           string
		allowedOrigin  string
		requestOrigin  string
		expectHeader   bool
	}{
		{
			name:          "Wildcard origin",
			allowedOrigin: "*",
			requestOrigin: "https://example.com",
			expectHeader:  true,
		},
		{
			name:          "Matching origin",
			allowedOrigin: "https://example.com",
			requestOrigin: "https://example.com",
			expectHeader:  true,
		},
		{
			name:          "Non-matching origin",
			allowedOrigin: "https://example.com",
			requestOrigin: "https://other.com",
			expectHeader:  false,
		},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg.AllowedOrigin = test.allowedOrigin
			
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Origin", test.requestOrigin)
			w := httptest.NewRecorder()
			
			h.allowCORS(w, req)
			
			originHeader := w.Header().Get("Access-Control-Allow-Origin")
			if test.expectHeader && originHeader == "" {
				t.Error("Expected CORS origin header to be set")
			} else if !test.expectHeader && originHeader != "" {
				t.Error("Expected CORS origin header not to be set")
			}
			
			// Check that required headers are always set
			methods := w.Header().Get("Access-Control-Allow-Methods")
			if methods == "" {
				t.Error("Expected Access-Control-Allow-Methods header to be set")
			}
			
			headers := w.Header().Get("Access-Control-Allow-Headers")
			if headers == "" {
				t.Error("Expected Access-Control-Allow-Headers header to be set")
			}
		})
	}
}