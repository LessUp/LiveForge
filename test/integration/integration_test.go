// +build integration

package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"live-webrtc-go/internal/api"
	"live-webrtc-go/internal/config"
	"live-webrtc-go/internal/sfu"
)

func setupIntegrationTest() (*api.HTTPHandlers, *config.Config, *httptest.Server) {
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
	h := api.NewHTTPHandlers(mgr, cfg)
	
	return h, cfg, nil
}

func TestIntegration_RoomLifecycle(t *testing.T) {
	h, _, _ := setupIntegrationTest()
	
	// Test room creation and listing
	req := httptest.NewRequest("GET", "/api/rooms", nil)
	w := httptest.NewRecorder()
	
	h.ServeRooms(w, req)
	
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}
	
	var rooms []sfu.RoomInfo
	err := json.NewDecoder(resp.Body).Decode(&rooms)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	if len(rooms) != 0 {
		t.Errorf("Expected 0 rooms initially, got %d", len(rooms))
	}
	
	// Create a room by attempting to publish (will fail due to invalid SDP)
	sdpOffer := "v=0\r\no=- 1234567890 1234567890 IN IP4 127.0.0.1\r\ns=-\r\nt=0 0\r\n"
	
	req2 := httptest.NewRequest("POST", "/api/whip/publish/test-room", bytes.NewReader([]byte(sdpOffer)))
	w2 := httptest.NewRecorder()
	
	h.ServeWHIPPublish(w2, req2, "test-room")
	
	// Room should be created even though publish failed
	req3 := httptest.NewRequest("GET", "/api/rooms", nil)
	w3 := httptest.NewRecorder()
	
	h.ServeRooms(w3, req3)
	
	resp3 := w3.Result()
	var roomsAfter []sfu.RoomInfo
	err = json.NewDecoder(resp3.Body).Decode(&roomsAfter)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	if len(roomsAfter) != 1 {
		t.Errorf("Expected 1 room after publish attempt, got %d", len(roomsAfter))
	}
	
	if roomsAfter[0].Name != "test-room" {
		t.Errorf("Expected room name 'test-room', got '%s'", roomsAfter[0].Name)
	}
}

func TestIntegration_Authentication(t *testing.T) {
	h, cfg, _ := setupIntegrationTest()
	
	// Set up authentication
	cfg.AuthToken = "test-auth-token"
	
	sdpOffer := "v=0\r\no=- 1234567890 1234567890 IN IP4 127.0.0.1\r\ns=-\r\nt=0 0\r\n"
	
	// Test without authentication
	req := httptest.NewRequest("POST", "/api/whip/publish/test-room", bytes.NewReader([]byte(sdpOffer)))
	w := httptest.NewRecorder()
	
	h.ServeWHIPPublish(w, req, "test-room")
	
	resp := w.Result()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401 without auth, got %d", resp.StatusCode)
	}
	
	// Test with authentication
	req2 := httptest.NewRequest("POST", "/api/whip/publish/test-room", bytes.NewReader([]byte(sdpOffer)))
	req2.Header.Set("X-Auth-Token", "test-auth-token")
	w2 := httptest.NewRecorder()
	
	h.ServeWHIPPublish(w2, req2, "test-room")
	
	resp2 := w2.Result()
	if resp2.StatusCode == http.StatusUnauthorized {
		t.Error("Expected authentication to succeed")
	}
}

func TestIntegration_RoomTokens(t *testing.T) {
	h, cfg, _ := setupIntegrationTest()
	
	// Set up room-specific tokens
	cfg.RoomTokens = map[string]string{
		"room1": "token1",
		"room2": "token2",
	}
	
	sdpOffer := "v=0\r\no=- 1234567890 1234567890 IN IP4 127.0.0.1\r\ns=-\r\nt=0 0\r\n"
	
	// Test room1 with correct token
	req := httptest.NewRequest("POST", "/api/whip/publish/room1", bytes.NewReader([]byte(sdpOffer)))
	req.Header.Set("X-Auth-Token", "token1")
	w := httptest.NewRecorder()
	
	h.ServeWHIPPublish(w, req, "room1")
	
	resp := w.Result()
	if resp.StatusCode == http.StatusUnauthorized {
		t.Error("Expected room1 authentication to succeed")
	}
	
	// Test room1 with wrong token
	req2 := httptest.NewRequest("POST", "/api/whip/publish/room1", bytes.NewReader([]byte(sdpOffer)))
	req2.Header.Set("X-Auth-Token", "wrong-token")
	w2 := httptest.NewRecorder()
	
	h.ServeWHIPPublish(w2, req2, "room1")
	
	resp2 := w2.Result()
	if resp2.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401 with wrong token, got %d", resp2.StatusCode)
	}
	
	// Test room2 with its token
	req3 := httptest.NewRequest("POST", "/api/whip/publish/room2", bytes.NewReader([]byte(sdpOffer)))
	req3.Header.Set("X-Auth-Token", "token2")
	w3 := httptest.NewRecorder()
	
	h.ServeWHIPPublish(w3, req3, "room2")
	
	resp3 := w3.Result()
	if resp3.StatusCode == http.StatusUnauthorized {
		t.Error("Expected room2 authentication to succeed")
	}
}

func TestIntegration_AdminCloseRoom(t *testing.T) {
	h, cfg, _ := setupIntegrationTest()
	
	// Set admin token
	cfg.AdminToken = "admin-token"
	
	// Create a room
	sdpOffer := "v=0\r\no=- 1234567890 1234567890 IN IP4 127.0.0.1\r\ns=-\r\nt=0 0\r\n"
	
	req := httptest.NewRequest("POST", "/api/whip/publish/test-room", bytes.NewReader([]byte(sdpOffer)))
	w := httptest.NewRecorder()
	
	h.ServeWHIPPublish(w, req, "test-room")
	
	// Verify room exists
	req2 := httptest.NewRequest("GET", "/api/rooms", nil)
	w2 := httptest.NewRecorder()
	
	h.ServeRooms(w2, req2)
	
	resp2 := w2.Result()
	var rooms []sfu.RoomInfo
	json.NewDecoder(resp2.Body).Decode(&rooms)
	
	if len(rooms) != 1 {
		t.Fatalf("Expected 1 room, got %d", len(rooms))
	}
	
	// Close room with admin auth
	req3 := httptest.NewRequest("POST", "/api/admin/rooms/test-room/close", nil)
	req3.Header.Set("Authorization", "Bearer admin-token")
	w3 := httptest.NewRecorder()
	
	h.ServeAdminCloseRoom(w3, req3, "test-room")
	
	resp3 := w3.Result()
	if resp3.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp3.StatusCode)
	}
	
	// Verify room is closed
	req4 := httptest.NewRequest("GET", "/api/rooms", nil)
	w4 := httptest.NewRecorder()
	
	h.ServeRooms(w4, req4)
	
	resp4 := w4.Result()
	var roomsAfter []sfu.RoomInfo
	json.NewDecoder(resp4.Body).Decode(&roomsAfter)
	
	if len(roomsAfter) != 0 {
		t.Errorf("Expected 0 rooms after closing, got %d", len(roomsAfter))
	}
}

func TestIntegration_RecordsList(t *testing.T) {
	h, cfg, _ := setupIntegrationTest()
	
	// Create temporary directory for records
	tempDir := t.TempDir()
	cfg.RecordDir = tempDir
	
	// Create test recording files
	testFiles := []struct {
		name    string
		content []byte
	}{
		{"test1.ivf", []byte("test ivf content 1")},
		{"test2.ivf", []byte("test ivf content 2")},
		{"test.ogg", []byte("test ogg content")},
		{"ignore.txt", []byte("should be ignored")},
	}
	
	for _, file := range testFiles {
		err := os.WriteFile(tempDir+"/"+file.name, file.content, 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", file.name, err)
		}
	}
	
	// Test records list
	req := httptest.NewRequest("GET", "/api/records", nil)
	w := httptest.NewRecorder()
	
	h.ServeRecordsList(w, req)
	
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	
	var records []map[string]interface{}
	err := json.NewDecoder(resp.Body).Decode(&records)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	// Should have 3 files (2 IVF + 1 OGG, ignore.txt should be filtered out)
	if len(records) != 3 {
		t.Errorf("Expected 3 recording files, got %d", len(records))
	}
	
	// Verify file names
	recordNames := make(map[string]bool)
	for _, record := range records {
		name, ok := record["name"].(string)
		if !ok {
			t.Error("Expected record to have 'name' field")
			continue
		}
		recordNames[name] = true
	}
	
	expectedFiles := []string{"test1.ivf", "test2.ivf", "test.ogg"}
	for _, expected := range expectedFiles {
		if !recordNames[expected] {
			t.Errorf("Expected file %s not found in records", expected)
		}
	}
}

func TestIntegration_CORS(t *testing.T) {
	h, cfg, _ := setupIntegrationTest()
	
	// Set specific allowed origin
	cfg.AllowedOrigin = "https://example.com"
	
	// Test with allowed origin
	req := httptest.NewRequest("OPTIONS", "/api/rooms", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()
	
	h.ServeRooms(w, req)
	
	resp := w.Result()
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", resp.StatusCode)
	}
	
	originHeader := resp.Header.Get("Access-Control-Allow-Origin")
	if originHeader != "https://example.com" {
		t.Errorf("Expected CORS origin to be 'https://example.com', got '%s'", originHeader)
	}
	
	// Test with disallowed origin
	req2 := httptest.NewRequest("OPTIONS", "/api/rooms", nil)
	req2.Header.Set("Origin", "https://malicious.com")
	w2 := httptest.NewRecorder()
	
	h.ServeRooms(w2, req2)
	
	resp2 := w2.Result()
	originHeader2 := resp2.Header.Get("Access-Control-Allow-Origin")
	if originHeader2 != "" {
		t.Errorf("Expected no CORS origin header for disallowed origin, got '%s'", originHeader2)
	}
}

func TestIntegration_RateLimiting(t *testing.T) {
	h, cfg, _ := setupIntegrationTest()
	
	// Enable rate limiting
	cfg.RateLimitRPS = 1.0
	cfg.RateLimitBurst = 1
	
	sdpOffer := "v=0\r\no=- 1234567890 1234567890 IN IP4 127.0.0.1\r\ns=-\r\nt=0 0\r\n"
	
	// First request should succeed (but fail due to invalid SDP)
	req := httptest.NewRequest("POST", "/api/whip/publish/test-room", bytes.NewReader([]byte(sdpOffer)))
	w := httptest.NewRecorder()
	
	h.ServeWHIPPublish(w, req, "test-room")
	
	resp := w.Result()
	if resp.StatusCode == http.StatusTooManyRequests {
		t.Error("Expected first request to not be rate limited")
	}
	
	// Immediate second request should be rate limited
	req2 := httptest.NewRequest("POST", "/api/whip/publish/test-room", bytes.NewReader([]byte(sdpOffer)))
	w2 := httptest.NewRecorder()
	
	h.ServeWHIPPublish(w2, req2, "test-room")
	
	resp2 := w2.Result()
	if resp2.StatusCode != http.StatusTooManyRequests {
		t.Errorf("Expected status 429 (rate limited), got %d", resp2.StatusCode)
	}
	
	// Wait for rate limit window to reset
	time.Sleep(2 * time.Second)
	
	// Third request should succeed again (but fail due to invalid SDP)
	req3 := httptest.NewRequest("POST", "/api/whip/publish/test-room", bytes.NewReader([]byte(sdpOffer)))
	w3 := httptest.NewRecorder()
	
	h.ServeWHIPPublish(w3, req3, "test-room")
	
	resp3 := w3.Result()
	if resp3.StatusCode == http.StatusTooManyRequests {
		t.Error("Expected third request to not be rate limited after waiting")
	}
}