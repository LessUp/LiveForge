// +build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

type TestServer struct {
	baseURL string
	client  *http.Client
}

func NewTestServer(baseURL string) *TestServer {
	return &TestServer{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (ts *TestServer) request(method, path string, body []byte, headers map[string]string) (*http.Response, error) {
	url := ts.baseURL + path
	
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}
	
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}
	
	// Set default headers
	if body != nil {
		req.Header.Set("Content-Type", "application/sdp")
	}
	
	// Apply custom headers
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	
	return ts.client.Do(req)
}

func TestE2E_ServerStartup(t *testing.T) {
	serverURL := os.Getenv("TEST_SERVER_URL")
	if serverURL == "" {
		serverURL = "http://localhost:8080"
	}
	
	ts := NewTestServer(serverURL)
	
	// Test health check endpoint
	resp, err := ts.request("GET", "/healthz", nil, nil)
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	
	if string(body) != "ok" {
		t.Errorf("Expected response body 'ok', got '%s'", string(body))
	}
}

func TestE2E_RoomsAPI(t *testing.T) {
	serverURL := os.Getenv("TEST_SERVER_URL")
	if serverURL == "" {
		serverURL = "http://localhost:8080"
	}
	
	ts := NewTestServer(serverURL)
	
	// Test rooms list endpoint
	resp, err := ts.request("GET", "/api/rooms", nil, nil)
	if err != nil {
		t.Fatalf("Failed to request rooms: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	
	var rooms []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&rooms)
	if err != nil {
		t.Fatalf("Failed to decode rooms response: %v", err)
	}
	
	if rooms == nil {
		t.Error("Expected rooms array, got nil")
	}
}

func TestE2E_RecordsAPI(t *testing.T) {
	serverURL := os.Getenv("TEST_SERVER_URL")
	if serverURL == "" {
		serverURL = "http://localhost:8080"
	}
	
	ts := NewTestServer(serverURL)
	
	// Test records list endpoint
	resp, err := ts.request("GET", "/api/records", nil, nil)
	if err != nil {
		t.Fatalf("Failed to request records: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	
	var records []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&records)
	if err != nil {
		t.Fatalf("Failed to decode records response: %v", err)
	}
	
	if records == nil {
		t.Error("Expected records array, got nil")
	}
}

func TestE2E_MetricsEndpoint(t *testing.T) {
	serverURL := os.Getenv("TEST_SERVER_URL")
	if serverURL == "" {
		serverURL = "http://localhost:8080"
	}
	
	ts := NewTestServer(serverURL)
	
	// Test metrics endpoint
	resp, err := ts.request("GET", "/metrics", nil, nil)
	if err != nil {
		t.Fatalf("Failed to request metrics: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/plain") {
		t.Errorf("Expected text/plain content type, got %s", contentType)
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read metrics response: %v", err)
	}
	
	// Check for expected metrics
	expectedMetrics := []string{
		"webrtc_rooms",
		"webrtc_subscribers",
		"webrtc_rtp_bytes_total",
		"webrtc_rtp_packets_total",
	}
	
	bodyStr := string(body)
	for _, metric := range expectedMetrics {
		if !strings.Contains(bodyStr, metric) {
			t.Errorf("Expected metric '%s' not found in response", metric)
		}
	}
}

func TestE2E_StaticFiles(t *testing.T) {
	serverURL := os.Getenv("TEST_SERVER_URL")
	if serverURL == "" {
		serverURL = "http://localhost:8080"
	}
	
	ts := NewTestServer(serverURL)
	
	// Test static files
	staticFiles := []string{
		"/web/index.html",
		"/web/publisher.html",
		"/web/player.html",
	}
	
	for _, file := range staticFiles {
		resp, err := ts.request("GET", file, nil, nil)
		if err != nil {
			t.Errorf("Failed to request %s: %v", file, err)
			continue
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200 for %s, got %d", file, resp.StatusCode)
		}
		
		contentType := resp.Header.Get("Content-Type")
		if !strings.Contains(contentType, "text/html") {
			t.Errorf("Expected HTML content type for %s, got %s", file, contentType)
		}
	}
}

func TestE2E_CORSHeaders(t *testing.T) {
	serverURL := os.Getenv("TEST_SERVER_URL")
	if serverURL == "" {
		serverURL = "http://localhost:8080"
	}
	
	ts := NewTestServer(serverURL)
	
	// Test CORS headers on API endpoints
	resp, err := ts.request("OPTIONS", "/api/rooms", nil, map[string]string{
		"Origin": "http://localhost:3000",
	})
	if err != nil {
		t.Fatalf("Failed to request with CORS: %v", err)
	}
	defer resp.Body.Close()
	
	// Check CORS headers
	allowedOrigin := resp.Header.Get("Access-Control-Allow-Origin")
	if allowedOrigin == "" {
		t.Error("Expected Access-Control-Allow-Origin header")
	}
	
	allowedMethods := resp.Header.Get("Access-Control-Allow-Methods")
	if allowedMethods == "" {
		t.Error("Expected Access-Control-Allow-Methods header")
	}
	
	allowedHeaders := resp.Header.Get("Access-Control-Allow-Headers")
	if allowedHeaders == "" {
		t.Error("Expected Access-Control-Allow-Headers header")
	}
}

func TestE2E_WebRTCPublishSubscribe(t *testing.T) {
	serverURL := os.Getenv("TEST_SERVER_URL")
	if serverURL == "" {
		serverURL = "http://localhost:8080"
	}
	
	ts := NewTestServer(serverURL)
	
	// Create a simple SDP offer (this won't be a valid WebRTC offer, but tests the API)
	sdpOffer := `v=0
o=- 1234567890 1234567890 IN IP4 127.0.0.1
s=-
t=0 0
m=audio 9 UDP/TLS/RTP/SAVPF 111
c=IN IP4 0.0.0.0
a=rtcp:9 IN IP4 0.0.0.0
a=ice-ufrag:test
a=ice-pwd:test123
a=ice-options:trickle
a=fingerprint:sha-256 00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00
a=setup:actpass
a=mid:0
a=sendrecv
a=rtcp-mux
a=rtpmap:111 opus/48000/2
a=fmtp:111 minptime=10;useinbandfec=1
`
	
	// Test publish endpoint
	resp, err := ts.request("POST", "/api/whip/publish/test-room", []byte(sdpOffer), nil)
	if err != nil {
		t.Fatalf("Failed to publish: %v", err)
	}
	defer resp.Body.Close()
	
	// We expect this to fail due to invalid SDP, but the API should be accessible
	if resp.StatusCode == http.StatusNotFound {
		t.Error("Expected endpoint to exist")
	}
	
	// Test subscribe endpoint
	resp2, err := ts.request("POST", "/api/whep/play/test-room", []byte(sdpOffer), nil)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	defer resp2.Body.Close()
	
	// We expect this to fail due to invalid SDP, but the API should be accessible
	if resp2.StatusCode == http.StatusNotFound {
		t.Error("Expected endpoint to exist")
	}
}

func TestE2E_AdminCloseRoom(t *testing.T) {
	serverURL := os.Getenv("TEST_SERVER_URL")
	if serverURL == "" {
		serverURL = "http://localhost:8080"
	}
	
	// Skip if admin token is not configured
	adminToken := os.Getenv("ADMIN_TOKEN")
	if adminToken == "" {
		t.Skip("ADMIN_TOKEN not configured, skipping admin test")
	}
	
	ts := NewTestServer(serverURL)
	
	// Create a room by attempting to publish
	sdpOffer := `v=0
o=- 1234567890 1234567890 IN IP4 127.0.0.1
s=-
t=0 0
m=audio 9 UDP/TLS/RTP/SAVPF 111
c=IN IP4 0.0.0.0
a=rtcp:9 IN IP4 0.0.0.0
a=ice-ufrag:test
a=ice-pwd:test123
a=ice-options:trickle
a=fingerprint:sha-256 00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00
a=setup:actpass
a=mid:0
a=sendrecv
a=rtcp-mux
a=rtpmap:111 opus/48000/2
a=fmtp:111 minptime=10;useinbandfec=1
`
	
	resp, err := ts.request("POST", "/api/whip/publish/test-room", []byte(sdpOffer), nil)
	if err != nil {
		t.Fatalf("Failed to create room: %v", err)
	}
	defer resp.Body.Close()
	
	// Close the room using admin endpoint
	resp2, err := ts.request("POST", "/api/admin/rooms/test-room/close", nil, map[string]string{
		"Authorization": "Bearer " + adminToken,
	})
	if err != nil {
		t.Fatalf("Failed to close room: %v", err)
	}
	defer resp2.Body.Close()
	
	if resp2.StatusCode != http.StatusOK && resp2.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 200 or 404, got %d", resp2.StatusCode)
	}
}

func TestE2E_ConcurrentRequests(t *testing.T) {
	serverURL := os.Getenv("TEST_SERVER_URL")
	if serverURL == "" {
		serverURL = "http://localhost:8080"
	}
	
	ts := NewTestServer(serverURL)
	
	// Test concurrent requests to various endpoints
	endpoints := []string{
		"/healthz",
		"/api/rooms",
		"/api/records",
		"/metrics",
	}
	
	numConcurrent := 10
	errors := make(chan error, numConcurrent*len(endpoints))
	
	for _, endpoint := range endpoints {
		for i := 0; i < numConcurrent; i++ {
			go func(url string) {
				resp, err := ts.request("GET", url, nil, nil)
				if err != nil {
					errors <- fmt.Errorf("request failed for %s: %v", url, err)
					return
				}
				defer resp.Body.Close()
				
				if resp.StatusCode != http.StatusOK {
					errors <- fmt.Errorf("unexpected status %d for %s", resp.StatusCode, url)
					return
				}
				errors <- nil
			}(endpoint)
		}
	}
	
	// Collect results
	errorCount := 0
	for i := 0; i < numConcurrent*len(endpoints); i++ {
		if err := <-errors; err != nil {
			t.Error(err)
			errorCount++
		}
	}
	
	if errorCount > 0 {
		t.Errorf("Encountered %d errors during concurrent testing", errorCount)
	}
}

func TestE2E_TimeoutHandling(t *testing.T) {
	serverURL := os.Getenv("TEST_SERVER_URL")
	if serverURL == "" {
		serverURL = "http://localhost:8080"
	}
	
	// Create client with very short timeout
	ts := &TestServer{
		baseURL: serverURL,
		client: &http.Client{
			Timeout: 1 * time.Millisecond, // Very short timeout
		},
	}
	
	// This should timeout
	resp, err := ts.request("GET", "/api/rooms", nil, nil)
	if err == nil {
		if resp != nil {
			resp.Body.Close()
		}
		t.Error("Expected timeout error")
	}
}