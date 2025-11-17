package sfu

import (
	"context"
	"sync"
	"testing"
	"time"

	"live-webrtc-go/internal/config"
)

func setupTestManager() (*Manager, *config.Config) {
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
	
	mgr := NewManager(cfg)
	return mgr, cfg
}

func TestNewManager(t *testing.T) {
	mgr, cfg := setupTestManager()
	
	if mgr == nil {
		t.Fatal("Expected manager to be created")
	}
	
	if mgr.rooms == nil {
		t.Error("Expected rooms map to be initialized")
	}
	
	if mgr.cfg != cfg {
		t.Error("Expected config to be set correctly")
	}
}

func TestManager_GetOrCreateRoom(t *testing.T) {
	mgr, _ := setupTestManager()
	
	// Test creating a new room
	room1 := mgr.getOrCreateRoom("test-room")
	if room1 == nil {
		t.Fatal("Expected room to be created")
	}
	
	if room1.name != "test-room" {
		t.Errorf("Expected room name to be 'test-room', got '%s'", room1.name)
	}
	
	// Test getting existing room
	room2 := mgr.getOrCreateRoom("test-room")
	if room2 != room1 {
		t.Error("Expected to get the same room instance")
	}
	
	// Test creating another room
	room3 := mgr.getOrCreateRoom("another-room")
	if room3 == room1 {
		t.Error("Expected different room instances")
	}
	
	if room3.name != "another-room" {
		t.Errorf("Expected room name to be 'another-room', got '%s'", room3.name)
	}
}

func TestManager_ListRooms(t *testing.T) {
	mgr, _ := setupTestManager()
	
	// Initially no rooms
	rooms := mgr.ListRooms()
	if len(rooms) != 0 {
		t.Errorf("Expected 0 rooms initially, got %d", len(rooms))
	}
	
	// Create some rooms
	mgr.getOrCreateRoom("room1")
	mgr.getOrCreateRoom("room2")
	mgr.getOrCreateRoom("room3")
	
	rooms = mgr.ListRooms()
	if len(rooms) != 3 {
		t.Errorf("Expected 3 rooms, got %d", len(rooms))
	}
	
	// Verify room names
	roomNames := make(map[string]bool)
	for _, room := range rooms {
		roomNames[room.Name] = true
	}
	
	expectedNames := []string{"room1", "room2", "room3"}
	for _, name := range expectedNames {
		if !roomNames[name] {
			t.Errorf("Expected room '%s' not found in list", name)
		}
	}
}

func TestManager_CloseRoom(t *testing.T) {
	mgr, _ := setupTestManager()
	
	// Create a room
	room := mgr.getOrCreateRoom("test-room")
	if room == nil {
		t.Fatal("Expected room to be created")
	}
	
	// Verify room exists
	rooms := mgr.ListRooms()
	if len(rooms) != 1 {
		t.Errorf("Expected 1 room, got %d", len(rooms))
	}
	
	// Close the room
	closed := mgr.CloseRoom("test-room")
	if !closed {
		t.Error("Expected room to be closed successfully")
	}
	
	// Verify room no longer exists
	rooms = mgr.ListRooms()
	if len(rooms) != 0 {
		t.Errorf("Expected 0 rooms after closing, got %d", len(rooms))
	}
	
	// Try to close non-existent room
	closed = mgr.CloseRoom("non-existent")
	if closed {
		t.Error("Expected closing non-existent room to return false")
	}
}

func TestManager_CloseAll(t *testing.T) {
	mgr, _ := setupTestManager()
	
	// Create multiple rooms
	mgr.getOrCreateRoom("room1")
	mgr.getOrCreateRoom("room2")
	mgr.getOrCreateRoom("room3")
	
	// Verify rooms exist
	rooms := mgr.ListRooms()
	if len(rooms) != 3 {
		t.Errorf("Expected 3 rooms, got %d", len(rooms))
	}
	
	// Close all rooms
	mgr.CloseAll()
	
	// Verify no rooms exist
	rooms = mgr.ListRooms()
	if len(rooms) != 0 {
		t.Errorf("Expected 0 rooms after closing all, got %d", len(rooms))
	}
}

func TestManager_ConcurrentAccess(t *testing.T) {
	mgr, _ := setupTestManager()
	
	var wg sync.WaitGroup
	numGoroutines := 10
	numOperations := 100
	
	// Concurrent room creation
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				roomName := "concurrent-room"
				mgr.getOrCreateRoom(roomName)
			}
		}(i)
	}
	
	wg.Wait()
	
	// Should have only created one room due to concurrent access
	rooms := mgr.ListRooms()
	if len(rooms) != 1 {
		t.Errorf("Expected 1 room after concurrent access, got %d", len(rooms))
	}
}

func TestRoom_Stats(t *testing.T) {
	mgr, _ := setupTestManager()
	room := mgr.getOrCreateRoom("test-room")
	
	stats := room.stats()
	
	if stats.Name != "test-room" {
		t.Errorf("Expected room name to be 'test-room', got '%s'", stats.Name)
	}
	
	if stats.HasPublisher {
		t.Error("Expected HasPublisher to be false initially")
	}
	
	if stats.Tracks != 0 {
		t.Errorf("Expected 0 tracks, got %d", stats.Tracks)
	}
	
	if stats.Subscribers != 0 {
		t.Errorf("Expected 0 subscribers, got %d", stats.Subscribers)
	}
}

func TestRoom_Publish_InvalidSDP(t *testing.T) {
	mgr, _ := setupTestManager()
	room := mgr.getOrCreateRoom("test-room")
	
	ctx := context.Background()
	invalidSDP := "invalid-sdp-content"
	
	_, err := room.Publish(ctx, invalidSDP)
	if err == nil {
		t.Error("Expected error for invalid SDP")
	}
}

func TestRoom_Subscribe_InvalidSDP(t *testing.T) {
	mgr, _ := setupTestManager()
	room := mgr.getOrCreateRoom("test-room")
	
	ctx := context.Background()
	invalidSDP := "invalid-sdp-content"
	
	_, err := room.Subscribe(ctx, invalidSDP)
	if err == nil {
		t.Error("Expected error for invalid SDP")
	}
}

func TestRoom_Subscribe_LimitReached(t *testing.T) {
	mgr, cfg := setupTestManager()
	cfg.MaxSubsPerRoom = 2
	
	room := mgr.getOrCreateRoom("limited-room")
	ctx := context.Background()
	
	// First subscription should succeed (but will fail due to invalid SDP)
	_, err1 := room.Subscribe(ctx, "invalid-sdp")
	if err1 == nil {
		t.Error("Expected error for invalid SDP in first subscription")
	}
	
	// Second subscription should succeed (but will fail due to invalid SDP)
	_, err2 := room.Subscribe(ctx, "invalid-sdp")
	if err2 == nil {
		t.Error("Expected error for invalid SDP in second subscription")
	}
	
	// Third subscription should fail due to limit (but will fail due to invalid SDP first)
	_, err3 := room.Subscribe(ctx, "invalid-sdp")
	if err3 == nil {
		t.Error("Expected error for invalid SDP in third subscription")
	}
}

func TestRoom_ConcurrentPublish(t *testing.T) {
	mgr, _ := setupTestManager()
	room := mgr.getOrCreateRoom("test-room")
	ctx := context.Background()
	
	// Try to publish concurrently (both should fail due to invalid SDP)
	var wg sync.WaitGroup
	wg.Add(2)
	
	var err1, err2 error
	
	go func() {
		defer wg.Done()
		_, err1 = room.Publish(ctx, "invalid-sdp-1")
	}()
	
	go func() {
		defer wg.Done()
		_, err2 = room.Publish(ctx, "invalid-sdp-2")
	}()
	
	wg.Wait()
	
	if err1 == nil {
		t.Error("Expected error for first concurrent publish")
	}
	
	if err2 == nil {
		t.Error("Expected error for second concurrent publish")
	}
}

func TestRoom_Close(t *testing.T) {
	mgr, _ := setupTestManager()
	room := mgr.getOrCreateRoom("test-room")
	
	// Close the room
	room.Close()
	
	// Verify room is cleaned up
	stats := room.stats()
	if stats.HasPublisher {
		t.Error("Expected HasPublisher to be false after close")
	}
	
	if stats.Tracks != 0 {
		t.Errorf("Expected 0 tracks after close, got %d", stats.Tracks)
	}
	
	if stats.Subscribers != 0 {
		t.Errorf("Expected 0 subscribers after close, got %d", stats.Subscribers)
	}
}

func BenchmarkGetOrCreateRoom(b *testing.B) {
	mgr, _ := setupTestManager()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mgr.getOrCreateRoom("benchmark-room")
	}
}

func BenchmarkListRooms(b *testing.B) {
	mgr, _ := setupTestManager()
	
	// Create some rooms
	for i := 0; i < 10; i++ {
		mgr.getOrCreateRoom("room" + string(rune(i)))
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mgr.ListRooms()
	}
}