// Package metrics 暴露 Prometheus 指标，用于教学场景下的基础观测：
// - 每房间 RTP 字节/包总量
// - 当前订阅者数量（Gauge）
// - 当前房间数量（Gauge）
package metrics

// 暴露 Prometheus 指标，方便排查每个房间的带宽与在线情况。

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

func SetRooms(n float64)          { Rooms.Set(n) }
func IncSubscribers(room string)  { Subscribers.WithLabelValues(room).Inc() }
func DecSubscribers(room string)  { Subscribers.WithLabelValues(room).Dec() }
func AddBytes(room string, n int) { RTPBytes.WithLabelValues(room).Add(float64(n)) }
func IncPackets(room string)      { RTPPackets.WithLabelValues(room).Inc() }
