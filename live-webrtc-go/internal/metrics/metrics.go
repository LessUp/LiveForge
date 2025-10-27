package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	RTPBytes = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "webrtc_rtp_bytes_total",
		Help: "Total RTP bytes received by room",
	}, []string{"room"})

	RTPPackets = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "webrtc_rtp_packets_total",
		Help: "Total RTP packets received by room",
	}, []string{"room"})

	Subscribers = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "webrtc_subscribers",
		Help: "Current subscribers per room",
	}, []string{"room"})

	Rooms = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "webrtc_rooms",
		Help: "Current rooms managed",
	})
)

func SetRooms(n float64) { Rooms.Set(n) }
func IncSubscribers(room string) { Subscribers.WithLabelValues(room).Inc() }
func DecSubscribers(room string) { Subscribers.WithLabelValues(room).Dec() }
func AddBytes(room string, n int) { RTPBytes.WithLabelValues(room).Add(float64(n)) }
func IncPackets(room string) { RTPPackets.WithLabelValues(room).Inc() }
