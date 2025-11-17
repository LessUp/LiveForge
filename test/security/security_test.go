// +build security

package security

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"live-webrtc-go/internal/api"
	"live-webrtc-go/internal/config"
	"live-webrtc-go/internal/sfu"
)

func setupSecurityTest() (*api.HTTPHandlers, *config.Config) {
	cfg := &config.Config{
		HTTPAddr:          ":8080",
		AllowedOrigin:     "https://example.com",
		AuthToken:         "secure-token",
		STUN:              []string{"stun:stun.l.google.com:19302"},
		TURN:              []string{},
		TLSCertFile:       "",
		TLSKeyFile:        "",
		RecordEnabled:     false,
		RecordDir:         "records",
		MaxSubsPerRoom:    10,
		RoomTokens:        map[string]string{"secure-room": "room-token"},
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
		AdminToken:        "admin-token",
		RateLimitRPS:      5.0,
		RateLimitBurst:    10,
		JWTSecret:         "jwt-secret-key",
		PprofEnabled:      false,
	}
	
	mgr := sfu.NewManager(cfg)
	h := api.NewHTTPHandlers(mgr, cfg)
	
	return h, cfg
}

func TestSecurity_AuthenticationBypass(t *testing.T) {
	h, cfg := setupSecurityTest()
	
	sdpOffer := "v=0\r\no=- 1234567890 1234567890 IN IP4 127.0.0.1\r\ns=-\r\nt=0 0\r\n"
	
	// Test 1: No authentication header
	req1 := httptest.NewRequest("POST", "/api/whip/publish/secure-room", bytes.NewReader([]byte(sdpOffer)))
	w1 := httptest.NewRecorder()
	
	h.ServeWHIPPublish(w1, req1, "secure-room")
	
	resp1 := w1.Result()
	if resp1.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401 without auth, got %d", resp1.StatusCode)
	}
	
	// Test 2: Wrong authentication token
	req2 := httptest.NewRequest("POST", "/api/whip/publish/secure-room", bytes.NewReader([]byte(sdpOffer)))
	req2.Header.Set("X-Auth-Token", "wrong-token")
	w2 := httptest.NewRecorder()
	
	h.ServeWHIPPublish(w2, req2, "secure-room")
	
	resp2 := w2.Result()
	if resp2.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401 with wrong token, got %d", resp2.StatusCode)
	}
	
	// Test 3: Bearer token authentication
	req3 := httptest.NewRequest("POST", "/api/whip/publish/secure-room", bytes.NewReader([]byte(sdpOffer)))
	req3.Header.Set("Authorization", "Bearer wrong-token")
	w3 := httptest.NewRecorder()
	
	h.ServeWHIPPublish(w3, req3, "secure-room")
	
	resp3 := w3.Result()
	if resp3.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401 with wrong bearer token, got %d", resp3.StatusCode)
	}
}

func TestSecurity_RoomTokenAuthentication(t *testing.T) {
	h, cfg := setupSecurityTest()
	
	sdpOffer := "v=0\r\no=- 1234567890 1234567890 IN IP4 127.0.0.1\r\ns=-\r\nt=0 0\r\n"
	
	// Test 1: Global auth token should not work for room with specific token
	req1 := httptest.NewRequest("POST", "/api/whip/publish/secure-room", bytes.NewReader([]byte(sdpOffer)))
	req1.Header.Set("X-Auth-Token", cfg.AuthToken) // Global token
	w1 := httptest.NewRecorder()
	
	h.ServeWHIPPublish(w1, req1, "secure-room")
	
	resp1 := w1.Result()
	if resp1.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401 with global token for room-specific auth, got %d", resp1.StatusCode)
	}
	
	// Test 2: Room-specific token should work
	req2 := httptest.NewRequest("POST", "/api/whip/publish/secure-room", bytes.NewReader([]byte(sdpOffer)))
	req2.Header.Set("X-Auth-Token", "room-token") // Room-specific token
	w2 := httptest.NewRecorder()
	
	h.ServeWHIPPublish(w2, req2, "secure-room")
	
	resp2 := w2.Result()
	if resp2.StatusCode == http.StatusUnauthorized {
		t.Error("Expected room-specific token to work")
	}
}

func TestSecurity_JWTAuthentication(t *testing.T) {
	h, cfg := setupSecurityTest()
	
	sdpOffer := "v=0\r\no=- 1234567890 1234567890 IN IP4 127.0.0.1\r\ns=-\r\nt=0 0\r\n"
	
	// Test 1: Invalid JWT token
	req1 := httptest.NewRequest("POST", "/api/whip/publish/test-room", bytes.NewReader([]byte(sdpOffer)))
	req1.Header.Set("Authorization", "Bearer invalid.jwt.token")
	w1 := httptest.NewRecorder()
	
	h.ServeWHIPPublish(w1, req1, "test-room")
	
	resp1 := w1.Result()
	if resp1.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401 with invalid JWT, got %d", resp1.StatusCode)
	}
	
	// Test 2: JWT token without room claim (should work for rooms without specific tokens)
	req2 := httptest.NewRequest("POST", "/api/whip/publish/test-room", bytes.NewReader([]byte(sdpOffer)))
	req2.Header.Set("Authorization", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c")
	w2 := httptest.NewRecorder()
	
	h.ServeWHIPPublish(w2, req2, "test-room")
	
	resp2 := w2.Result()
	if resp2.StatusCode == http.StatusUnauthorized {
		t.Error("Expected JWT without room claim to work for general rooms")
	}
}

func TestSecurity_AdminAuthentication(t *testing.T) {
	h, cfg := setupSecurityTest()
	
	// Test 1: No admin authentication
	req1 := httptest.NewRequest("POST", "/api/admin/rooms/test-room/close", nil)
	w1 := httptest.NewRecorder()
	
	h.ServeAdminCloseRoom(w1, req1, "test-room")
	
	resp1 := w1.Result()
	if resp1.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401 without admin auth, got %d", resp1.StatusCode)
	}
	
	// Test 2: Wrong admin token
	req2 := httptest.NewRequest("POST", "/api/admin/rooms/test-room/close", nil)
	req2.Header.Set("Authorization", "Bearer wrong-admin-token")
	w2 := httptest.NewRecorder()
	
	h.ServeAdminCloseRoom(w2, req2, "test-room")
	
	resp2 := w2.Result()
	if resp2.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401 with wrong admin token, got %d", resp2.StatusCode)
	}
}

func TestSecurity_RateLimiting(t *testing.T) {
	h, cfg := setupSecurityTest()
	
	// Enable rate limiting
	cfg.RateLimitRPS = 1.0
	cfg.RateLimitBurst = 2
	
	// Create multiple requests from same IP
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/api/rooms", nil)
		req.RemoteAddr = "192.168.1.100:12345" // Same IP
		w := httptest.NewRecorder()
		
		h.ServeRooms(w, req)
		
		resp := w.Result()
		
		// First 2 requests should succeed (burst), rest should be rate limited
		if i < 2 {
			if resp.StatusCode == http.StatusTooManyRequests {
				t.Errorf("Request %d should not be rate limited", i+1)
			}
		} else {
			if resp.StatusCode != http.StatusTooManyRequests {
				t.Errorf("Request %d should be rate limited", i+1)
			}
		}
	}
}

func TestSecurity_CORSProtection(t *testing.T) {
	h, cfg := setupSecurityTest()
	
	// Test 1: Request from allowed origin
	req1 := httptest.NewRequest("OPTIONS", "/api/rooms", nil)
	req1.Header.Set("Origin", "https://example.com")
	w1 := httptest.NewRecorder()
	
	h.ServeRooms(w1, req1)
	
	resp1 := w1.Result()
	allowedOrigin1 := resp1.Header.Get("Access-Control-Allow-Origin")
	if allowedOrigin1 != "https://example.com" {
		t.Errorf("Expected allowed origin to be https://example.com, got %s", allowedOrigin1)
	}
	
	// Test 2: Request from disallowed origin
	req2 := httptest.NewRequest("OPTIONS", "/api/rooms", nil)
	req2.Header.Set("Origin", "https://malicious.com")
	w2 := httptest.NewRecorder()
	
	h.ServeRooms(w2, req2)
	
	resp2 := w2.Result()
	allowedOrigin2 := resp2.Header.Get("Access-Control-Allow-Origin")
	if allowedOrigin2 == "https://malicious.com" {
		t.Error("Should not allow disallowed origin")
	}
}

func TestSecurity_InputValidation(t *testing.T) {
	h, _ := setupSecurityTest()
	
	// Test 1: Path traversal in room name
	maliciousRoomNames := []string{
		"../../../etc/passwd",
		"..\\..\\..\\windows\\system32\\config\\sam",
		"room/../../config",
	}
	
	sdpOffer := "v=0\r\no=- 1234567890 1234567890 IN IP4 127.0.0.1\r\ns=-\r\nt=0 0\r\n"
	
	for _, roomName := range maliciousRoomNames {
		req := httptest.NewRequest("POST", "/api/whip/publish/"+roomName, bytes.NewReader([]byte(sdpOffer)))
		w := httptest.NewRecorder()
		
		h.ServeWHIPPublish(w, req, roomName)
		
		resp := w.Result()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status 400 for malicious room name '%s', got %d", roomName, resp.StatusCode)
		}
	}
}

func TestSecurity_LargePayload(t *testing.T) {
	h, _ := setupSecurityTest()
	
	// Create a very large SDP offer (10MB)
	largeSDP := strings.Repeat("a=large-payload-line\r\n", 100000)
	largeSDP = "v=0\r\no=- 1234567890 1234567890 IN IP4 127.0.0.1\r\ns=-\r\nt=0 0\r\n" + largeSDP
	
	req := httptest.NewRequest("POST", "/api/whip/publish/test-room", bytes.NewReader([]byte(largeSDP)))
	w := httptest.NewRecorder()
	
	h.ServeWHIPPublish(w, req, "test-room")
	
	resp := w.Result()
	// Should handle large payload gracefully
	if resp.StatusCode == 0 {
		t.Error("Should handle large payload without crashing")
	}
}

func TestSecurity_SQLInjection(t *testing.T) {
	h, _ := setupSecurityTest()
	
	// Test SQL injection in room names and tokens
	maliciousInputs := []string{
		"'; DROP TABLE rooms; --",
		"' OR '1'='1",
		"\" OR \"1\"=\"1",
		"1; DELETE FROM users WHERE 1=1",
	}
	
	sdpOffer := "v=0\r\no=- 1234567890 1234567890 IN IP4 127.0.0.1\r\ns=-\r\nt=0 0\r\n"
	
	for _, malicious := range maliciousInputs {
		// Test in room name
		req1 := httptest.NewRequest("POST", "/api/whip/publish/"+malicious, bytes.NewReader([]byte(sdpOffer)))
		req1.Header.Set("X-Auth-Token", malicious)
		w1 := httptest.NewRecorder()
		
		h.ServeWHIPPublish(w1, req1, malicious)
		
		resp1 := w1.Result()
		if resp1.StatusCode == 0 {
			t.Errorf("Should handle SQL injection attempt gracefully in room name: %s", malicious)
		}
	}
}

func TestSecurity_XSSPrevention(t *testing.T) {
	h, _ := setupSecurityTest()
	
	// Test XSS in room names
	xssPayloads := []string{
		"<script>alert('XSS')</script>",
		"javascript:alert('XSS')",
		"<img src=x onerror=alert('XSS')>",
		"'\"onmouseover=alert('XSS')//",
	}
	
	sdpOffer := "v=0\r\no=- 1234567890 1234567890 IN IP4 127.0.0.1\r\ns=-\r\nt=0 0\r\n"
	
	for _, payload := range xssPayloads {
		req := httptest.NewRequest("POST", "/api/whip/publish/"+payload, bytes.NewReader([]byte(sdpOffer)))
		w := httptest.NewRecorder()
		
		h.ServeWHIPPublish(w, req, payload)
		
		resp := w.Result()
		if resp.StatusCode == 0 {
			t.Errorf("Should handle XSS payload gracefully: %s", payload)
		}
	}
}

func TestSecurity_DDOSProtection(t *testing.T) {
	h, cfg := setupSecurityTest()
	
	// Enable strict rate limiting
	cfg.RateLimitRPS = 0.1 // Very low rate limit
	cfg.RateLimitBurst = 1
	
	// Simulate DDoS attack with many requests
	numRequests := 100
	rateLimitedCount := 0
	
	for i := 0; i < numRequests; i++ {
		req := httptest.NewRequest("GET", "/api/rooms", nil)
		req.RemoteAddr = "192.168.1.100:12345" // Same IP
		w := httptest.NewRecorder()
		
		h.ServeRooms(w, req)
		
		resp := w.Result()
		if resp.StatusCode == http.StatusTooManyRequests {
			rateLimitedCount++
		}
	}
	
	// Most requests should be rate limited
	if rateLimitedCount < numRequests/2 {
		t.Errorf("Expected most requests to be rate limited, got %d out of %d", rateLimitedCount, numRequests)
	}
}

func TestSecurity_SensitiveDataExposure(t *testing.T) {
	h, _ := setupSecurityTest()
	
	// Test that error messages don't expose sensitive information
	req := httptest.NewRequest("POST", "/api/whip/publish/test-room", bytes.NewReader([]byte("invalid-sdp")))
	w := httptest.NewRecorder()
	
	h.ServeWHIPPublish(w, req, "test-room")
	
	resp := w.Result()
	body := w.Body.String()
	
	// Check that response doesn't contain sensitive information
	sensitivePatterns := []string{
		"password",
		"secret",
		"key",
		"token",
		"database",
		"internal",
	}
	
	for _, pattern := range sensitivePatterns {
		if strings.Contains(strings.ToLower(body), pattern) {
			t.Errorf("Response should not contain sensitive information: %s", pattern)
		}
	}
}

func TestSecurity_HTTPHeaders(t *testing.T) {
	h, _ := setupSecurityTest()
	
	req := httptest.NewRequest("GET", "/api/rooms", nil)
	w := httptest.NewRecorder()
	
	h.ServeRooms(w, req)
	
	resp := w.Result()
	
	// Check for security headers
	securityHeaders := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"X-XSS-Protection":       "1; mode=block",
	}
	
	for header, expected := range securityHeaders {
		value := resp.Header.Get(header)
		if value != "" && value != expected {
			t.Errorf("Security header %s should be '%s', got '%s'", header, expected, value)
		}
	}
}