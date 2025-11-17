// +build performance

package performance

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"live-webrtc-go/internal/api"
	"live-webrtc-go/internal/config"
	"live-webrtc-go/internal/sfu"
)

func setupPerformanceTest() (*api.HTTPHandlers, *config.Config) {
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
	
	return h, cfg
}

func BenchmarkRoomCreation(b *testing.B) {
	h, _ := setupPerformanceTest()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		roomName := fmt.Sprintf("benchmark-room-%d", i)
		req := httptest.NewRequest("POST", "/api/whip/publish/"+roomName, 
			bytes.NewReader([]byte("v=0\r\no=- 1234567890 1234567890 IN IP4 127.0.0.1\r\ns=-\r\nt=0 0\r\n")))
		w := httptest.NewRecorder()
		
		h.ServeWHIPPublish(w, req, roomName)
	}
}

func BenchmarkRoomListing(b *testing.B) {
	h, _ := setupPerformanceTest()
	
	// Create some rooms first
	for i := 0; i < 100; i++ {
		roomName := fmt.Sprintf("setup-room-%d", i)
		req := httptest.NewRequest("POST", "/api/whip/publish/"+roomName,
			bytes.NewReader([]byte("v=0\r\no=- 1234567890 1234567890 IN IP4 127.0.0.1\r\ns=-\r\nt=0 0\r\n")))
		w := httptest.NewRecorder()
		
		h.ServeWHIPPublish(w, req, roomName)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/rooms", nil)
		w := httptest.NewRecorder()
		
		h.ServeRooms(w, req)
	}
}

func BenchmarkConcurrentRequests(b *testing.B) {
	h, _ := setupPerformanceTest()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		counter := 0
		for pb.Next() {
			roomName := fmt.Sprintf("concurrent-room-%d", counter)
			req := httptest.NewRequest("GET", "/api/rooms", nil)
			w := httptest.NewRecorder()
			
			h.ServeRooms(w, req)
			counter++
		}
	})
}

func BenchmarkPublishSubscribeCycle(b *testing.B) {
	h, _ := setupPerformanceTest()
	
	sdpOffer := []byte(`v=0
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
`)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		roomName := fmt.Sprintf("cycle-room-%d", i)
		
		// Publish
		req1 := httptest.NewRequest("POST", "/api/whip/publish/"+roomName, bytes.NewReader(sdpOffer))
		w1 := httptest.NewRecorder()
		h.ServeWHIPPublish(w1, req1, roomName)
		
		// Subscribe
		req2 := httptest.NewRequest("POST", "/api/whep/play/"+roomName, bytes.NewReader(sdpOffer))
		w2 := httptest.NewRecorder()
		h.ServeWHEPPlay(w2, req2, roomName)
	}
}

func TestPerformance_HighConcurrency(t *testing.T) {
	h, _ := setupPerformanceTest()
	
	numRequests := 1000
	numWorkers := 50
	
	// Channel to distribute work
	work := make(chan int, numRequests)
	var successCount int64
	var errorCount int64
	
	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range work {
				req := httptest.NewRequest("GET", "/api/rooms", nil)
				w := httptest.NewRecorder()
				
				h.ServeRooms(w, req)
				
				if w.Result().StatusCode == http.StatusOK {
					atomic.AddInt64(&successCount, 1)
				} else {
					atomic.AddInt64(&errorCount, 1)
				}
			}
		}()
	}
	
	// Start timer
	start := time.Now()
	
	// Distribute work
	for i := 0; i < numRequests; i++ {
		work <- i
	}
	close(work)
	
	// Wait for completion
	wg.Wait()
	
	elapsed := time.Since(start)
	
	t.Logf("High concurrency test completed in %v", elapsed)
	t.Logf("Total requests: %d", numRequests)
	t.Logf("Successful: %d", successCount)
	t.Logf("Errors: %d", errorCount)
	t.Logf("Requests per second: %.2f", float64(numRequests)/elapsed.Seconds())
	
	if errorCount > 0 {
		t.Errorf("Expected 0 errors, got %d", errorCount)
	}
	
	if successCount != int64(numRequests) {
		t.Errorf("Expected all requests to succeed, got %d successes", successCount)
	}
}

func TestPerformance_MemoryUsage(t *testing.T) {
	h, _ := setupPerformanceTest()
	
	// Create many rooms to test memory usage
	numRooms := 100
	
	// Measure memory before
	var m1 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)
	
	// Create rooms
	for i := 0; i < numRooms; i++ {
		roomName := fmt.Sprintf("memory-test-room-%d", i)
		req := httptest.NewRequest("POST", "/api/whip/publish/"+roomName,
			bytes.NewReader([]byte("v=0\r\no=- 1234567890 1234567890 IN IP4 127.0.0.1\r\ns=-\r\nt=0 0\r\n")))
		w := httptest.NewRecorder()
		
		h.ServeWHIPPublish(w, req, roomName)
	}
	
	// Measure memory after
	var m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m2)
	
	memoryIncrease := m2.Alloc - m1.Alloc
	memoryPerRoom := memoryIncrease / uint64(numRooms)
	
	t.Logf("Memory usage test:")
	t.Logf("Rooms created: %d", numRooms)
	t.Logf("Memory increase: %d bytes", memoryIncrease)
	t.Logf("Memory per room: %d bytes", memoryPerRoom)
	t.Logf("Memory per room: %.2f KB", float64(memoryPerRoom)/1024)
	
	// Check if memory usage is reasonable (less than 100KB per room)
	if memoryPerRoom > 100*1024 {
		t.Errorf("Memory usage per room is too high: %d bytes", memoryPerRoom)
	}
}

func TestPerformance_ResponseTime(t *testing.T) {
	h, _ := setupPerformanceTest()
	
	numRequests := 100
	responseTimes := make([]time.Duration, numRequests)
	
	for i := 0; i < numRequests; i++ {
		start := time.Now()
		
		req := httptest.NewRequest("GET", "/api/rooms", nil)
		w := httptest.NewRecorder()
		
		h.ServeRooms(w, req)
		
		elapsed := time.Since(start)
		responseTimes[i] = elapsed
	}
	
	// Calculate statistics
	var total time.Duration
	var min, max time.Duration
	
	if len(responseTimes) > 0 {
		min = responseTimes[0]
		max = responseTimes[0]
	}
	
	for _, rt := range responseTimes {
		total += rt
		if rt < min {
			min = rt
		}
		if rt > max {
			max = rt
		}
	}
	
	avg := total / time.Duration(numRequests)
	
	t.Logf("Response time analysis:")
	t.Logf("Total requests: %d", numRequests)
	t.Logf("Average response time: %v", avg)
	t.Logf("Min response time: %v", min)
	t.Logf("Max response time: %v", max)
	t.Logf("Total time: %v", total)
	
	// Check if average response time is reasonable (less than 10ms)
	if avg > 10*time.Millisecond {
		t.Errorf("Average response time is too high: %v", avg)
	}
}

func TestPerformance_RateLimiting(t *testing.T) {
	h, cfg := setupPerformanceTest()
	
	// Enable rate limiting
	cfg.RateLimitRPS = 10.0
	cfg.RateLimitBurst = 5
	
	numRequests := 50
	rateLimitedCount := 0
	
	start := time.Now()
	
	for i := 0; i < numRequests; i++ {
		req := httptest.NewRequest("GET", "/api/rooms", nil)
		req.RemoteAddr = "127.0.0.1:12345" // Same IP for rate limiting
		w := httptest.NewRecorder()
		
		h.ServeRooms(w, req)
		
		if w.Result().StatusCode == http.StatusTooManyRequests {
			rateLimitedCount++
		}
	}
	
	elapsed := time.Since(start)
	
	t.Logf("Rate limiting test:")
	t.Logf("Total requests: %d", numRequests)
	t.Logf("Rate limited: %d", rateLimitedCount)
	t.Logf("Time elapsed: %v", elapsed)
	t.Logf("Requests per second: %.2f", float64(numRequests)/elapsed.Seconds())
	
	// Should have some rate limiting
	if rateLimitedCount == 0 {
		t.Error("Expected some requests to be rate limited")
	}
	
	if rateLimitedCount == numRequests {
		t.Error("Expected some requests to succeed")
	}
}

func BenchmarkAuthCheck(b *testing.B) {
	h, cfg := setupPerformanceTest()
	
	// Set up authentication
	cfg.AuthToken = "benchmark-token"
	cfg.RoomTokens["benchmark-room"] = "room-token"
	cfg.JWTSecret = "benchmark-jwt-secret"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/api/whip/publish/benchmark-room", nil)
		req.Header.Set("X-Auth-Token", "room-token")
		w := httptest.NewRecorder()
		
		h.ServeWHIPPublish(w, req, "benchmark-room")
	}
}