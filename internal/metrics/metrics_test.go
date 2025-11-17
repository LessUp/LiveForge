package metrics

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestMetrics_InitialValues(t *testing.T) {
	// Test initial values of metrics
	
	// Test rooms gauge
	roomsValue := testutil.ToFloat64(Rooms)
	if roomsValue != 0 {
		t.Errorf("Expected initial rooms value to be 0, got %f", roomsValue)
	}
	
	// Test RTP bytes counter
	rtpBytesValue := testutil.ToFloat64(RTPBytes.WithLabelValues("test-room"))
	if rtpBytesValue != 0 {
		t.Errorf("Expected initial RTP bytes value to be 0, got %f", rtpBytesValue)
	}
	
	// Test RTP packets counter
	rtpPacketsValue := testutil.ToFloat64(RTPPackets.WithLabelValues("test-room"))
	if rtpPacketsValue != 0 {
		t.Errorf("Expected initial RTP packets value to be 0, got %f", rtpPacketsValue)
	}
	
	// Test subscribers gauge
	subscribersValue := testutil.ToFloat64(Subscribers.WithLabelValues("test-room"))
	if subscribersValue != 0 {
		t.Errorf("Expected initial subscribers value to be 0, got %f", subscribersValue)
	}
}

func TestSetRooms(t *testing.T) {
	// Set rooms to 5
	SetRooms(5)
	
	roomsValue := testutil.ToFloat64(Rooms)
	if roomsValue != 5 {
		t.Errorf("Expected rooms value to be 5, got %f", roomsValue)
	}
	
	// Set rooms to 10
	SetRooms(10)
	
	roomsValue = testutil.ToFloat64(Rooms)
	if roomsValue != 10 {
		t.Errorf("Expected rooms value to be 10, got %f", roomsValue)
	}
	
	// Reset to 0
	SetRooms(0)
	
	roomsValue = testutil.ToFloat64(Rooms)
	if roomsValue != 0 {
		t.Errorf("Expected rooms value to be 0, got %f", roomsValue)
	}
}

func TestIncSubscribers(t *testing.T) {
	room := "test-room"
	
	// Increment subscribers
	IncSubscribers(room)
	
	subscribersValue := testutil.ToFloat64(Subscribers.WithLabelValues(room))
	if subscribersValue != 1 {
		t.Errorf("Expected subscribers value to be 1, got %f", subscribersValue)
	}
	
	// Increment again
	IncSubscribers(room)
	
	subscribersValue = testutil.ToFloat64(Subscribers.WithLabelValues(room))
	if subscribersValue != 2 {
		t.Errorf("Expected subscribers value to be 2, got %f", subscribersValue)
	}
}

func TestDecSubscribers(t *testing.T) {
	room := "test-room"
	
	// First increment to 3
	IncSubscribers(room)
	IncSubscribers(room)
	IncSubscribers(room)
	
	subscribersValue := testutil.ToFloat64(Subscribers.WithLabelValues(room))
	if subscribersValue != 3 {
		t.Errorf("Expected subscribers value to be 3, got %f", subscribersValue)
	}
	
	// Decrement
	DecSubscribers(room)
	
	subscribersValue = testutil.ToFloat64(Subscribers.WithLabelValues(room))
	if subscribersValue != 2 {
		t.Errorf("Expected subscribers value to be 2, got %f", subscribersValue)
	}
	
	// Decrement again
	DecSubscribers(room)
	
	subscribersValue = testutil.ToFloat64(Subscribers.WithLabelValues(room))
	if subscribersValue != 1 {
		t.Errorf("Expected subscribers value to be 1, got %f", subscribersValue)
	}
}

func TestAddBytes(t *testing.T) {
	room := "test-room"
	
	// Add 1000 bytes
	AddBytes(room, 1000)
	
	rtpBytesValue := testutil.ToFloat64(RTPBytes.WithLabelValues(room))
	if rtpBytesValue != 1000 {
		t.Errorf("Expected RTP bytes value to be 1000, got %f", rtpBytesValue)
	}
	
	// Add another 500 bytes
	AddBytes(room, 500)
	
	rtpBytesValue = testutil.ToFloat64(RTPBytes.WithLabelValues(room))
	if rtpBytesValue != 1500 {
		t.Errorf("Expected RTP bytes value to be 1500, got %f", rtpBytesValue)
	}
}

func TestIncPackets(t *testing.T) {
	room := "test-room"
	
	// Increment packets
	IncPackets(room)
	
	rtpPacketsValue := testutil.ToFloat64(RTPPackets.WithLabelValues(room))
	if rtpPacketsValue != 1 {
		t.Errorf("Expected RTP packets value to be 1, got %f", rtpPacketsValue)
	}
	
	// Increment again
	IncPackets(room)
	
	rtpPacketsValue = testutil.ToFloat64(RTPPackets.WithLabelValues(room))
	if rtpPacketsValue != 2 {
		t.Errorf("Expected RTP packets value to be 2, got %f", rtpPacketsValue)
	}
}

func TestMetrics_ConcurrentAccess(t *testing.T) {
	// Test concurrent access to metrics
	done := make(chan bool)
	
	// Start multiple goroutines updating metrics
	for i := 0; i < 10; i++ {
		go func(id int) {
			room := "concurrent-room"
			for j := 0; j < 100; j++ {
				IncSubscribers(room)
				AddBytes(room, 100)
				IncPackets(room)
				if j%10 == 0 {
					DecSubscribers(room)
				}
			}
			done <- true
		}(i)
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
	
	// Verify metrics are consistent (should not panic or crash)
	subscribersValue := testutil.ToFloat64(Subscribers.WithLabelValues("concurrent-room"))
	if subscribersValue < 0 {
		t.Errorf("Subscribers value should not be negative: %f", subscribersValue)
	}
	
	rtpBytesValue := testutil.ToFloat64(RTPBytes.WithLabelValues("concurrent-room"))
	if rtpBytesValue <= 0 {
		t.Errorf("RTP bytes value should be positive: %f", rtpBytesValue)
	}
	
	rtpPacketsValue := testutil.ToFloat64(RTPPackets.WithLabelValues("concurrent-room"))
	if rtpPacketsValue <= 0 {
		t.Errorf("RTP packets value should be positive: %f", rtpPacketsValue)
	}
}

func TestMetrics_Labels(t *testing.T) {
	// Test that metrics work with different room labels
	rooms := []string{"room1", "room2", "room3"}
	
	for _, room := range rooms {
		SetRooms(float64(len(rooms)))
		IncSubscribers(room)
		AddBytes(room, 1000)
		IncPackets(room)
	}
	
	// Verify each room has its own metrics
	for _, room := range rooms {
		subscribersValue := testutil.ToFloat64(Subscribers.WithLabelValues(room))
		if subscribersValue != 1 {
			t.Errorf("Expected subscribers for room %s to be 1, got %f", room, subscribersValue)
		}
		
		rtpBytesValue := testutil.ToFloat64(RTPBytes.WithLabelValues(room))
		if rtpBytesValue != 1000 {
			t.Errorf("Expected RTP bytes for room %s to be 1000, got %f", room, rtpBytesValue)
		}
		
		rtpPacketsValue := testutil.ToFloat64(RTPPackets.WithLabelValues(room))
		if rtpPacketsValue != 1 {
			t.Errorf("Expected RTP packets for room %s to be 1, got %f", room, rtpPacketsValue)
		}
	}
	
	// Verify rooms gauge
	roomsValue := testutil.ToFloat64(Rooms)
	if roomsValue != float64(len(rooms)) {
		t.Errorf("Expected rooms value to be %d, got %f", len(rooms), roomsValue)
	}
}

func BenchmarkIncSubscribers(b *testing.B) {
	room := "benchmark-room"
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		IncSubscribers(room)
	}
}

func BenchmarkAddBytes(b *testing.B) {
	room := "benchmark-room"
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		AddBytes(room, 1024)
	}
}

func BenchmarkIncPackets(b *testing.B) {
	room := "benchmark-room"
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		IncPackets(room)
	}
}