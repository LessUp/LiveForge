// Package sfu 提供轻量级房间与轨道分发的教学实现。
package sfu

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media/ivfwriter"
	"github.com/pion/webrtc/v3/pkg/media/oggwriter"
	"live-webrtc-go/internal/config"
	"live-webrtc-go/internal/metrics"
	"live-webrtc-go/internal/uploader"
)

// Manager 负责跟踪所有房间的生命周期，提供 Publish/Subscribe 入口。
type Manager struct {
	mu    sync.RWMutex
	rooms map[string]*Room
	cfg   *config.Config
}

// CloseRoom 主动关闭指定房间并更新房间数量指标。
func (m *Manager) CloseRoom(name string) bool {
	m.mu.Lock()
	r, ok := m.rooms[name]
	if ok {
		delete(m.rooms, name)
	}
	n := len(m.rooms)
	m.mu.Unlock()
	if ok {
		r.Close()
		metrics.SetRooms(float64(n))
	}
	return ok
}

// CloseAll 在服务退出时关闭所有房间，避免 WebRTC 连接泄漏。
func (m *Manager) CloseAll() {
	m.mu.Lock()
	rooms := make([]*Room, 0, len(m.rooms))
	for _, r := range m.rooms {
		rooms = append(rooms, r)
	}
	m.rooms = make(map[string]*Room)
	m.mu.Unlock()
	for _, r := range rooms {
		r.Close()
	}
	metrics.SetRooms(0)
}

// NewManager 创建一个房间管理器。
func NewManager(c *config.Config) *Manager {
	return &Manager{rooms: make(map[string]*Room), cfg: c}
}

// getOrCreateRoom 获取或创建房间，首次创建时更新房间计数指标。
func (m *Manager) getOrCreateRoom(name string) *Room {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.rooms[name]
	if !ok {
		r = NewRoom(name, m)
		m.rooms[name] = r
		metrics.SetRooms(float64(len(m.rooms)))
	}
	return r
}

// Publish 根据房间名将 SDP Offer 交给对应 Room 处理，返回 SDP Answer。
func (m *Manager) Publish(ctx context.Context, roomName, offerSDP string) (string, error) {
	r := m.getOrCreateRoom(roomName)
	return r.Publish(ctx, offerSDP)
}

// Subscribe 根据房间名将 SDP Offer 交给对应 Room 处理，返回 SDP Answer。
func (m *Manager) Subscribe(ctx context.Context, roomName, offerSDP string) (string, error) {
	r := m.getOrCreateRoom(roomName)
	return r.Subscribe(ctx, offerSDP)
}

type RoomInfo struct {
	Name         string
	HasPublisher bool
	Tracks       int
	Subscribers  int
}

func (m *Manager) ListRooms() []RoomInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]RoomInfo, 0, len(m.rooms))
	for _, r := range m.rooms {
		out = append(out, r.stats())
	}
	return out
}

// Room 表示一个 SFU 房间，维护发布者、订阅者与轨道 fanout。
type Room struct {
	name       string
	mu         sync.RWMutex
	publisher  *webrtc.PeerConnection
	trackFeeds map[string]*trackFanout // key: track ID
	subs       map[*webrtc.PeerConnection]struct{}
	mgr        *Manager
}

// NewRoom 初始化房间默认状态。
func NewRoom(name string, m *Manager) *Room {
	return &Room{
		name:       name,
		trackFeeds: make(map[string]*trackFanout),
		subs:       make(map[*webrtc.PeerConnection]struct{}),
		mgr:        m,
	}
}

// iceConfig 生成 ICE 配置，优先使用配置中的 STUN/TURN。
func (r *Room) iceConfig() webrtc.Configuration {
	var servers []webrtc.ICEServer
	if r.mgr != nil && r.mgr.cfg != nil {
		if len(r.mgr.cfg.STUN) > 0 {
			servers = append(servers, webrtc.ICEServer{URLs: r.mgr.cfg.STUN})
		}
		if len(r.mgr.cfg.TURN) > 0 {
			s := webrtc.ICEServer{URLs: r.mgr.cfg.TURN}
			if r.mgr.cfg.TURNUsername != "" || r.mgr.cfg.TURNPassword != "" {
				s.Username = r.mgr.cfg.TURNUsername
				s.Credential = r.mgr.cfg.TURNPassword
				s.CredentialType = webrtc.ICECredentialTypePassword
			}
			servers = append(servers, s)
		}
	}
	if len(servers) == 0 {
		servers = []webrtc.ICEServer{{URLs: []string{"stun:stun.l.google.com:19302"}}}
	}
	return webrtc.Configuration{ICEServers: servers}
}

// Publish 接收主播的 SDP Offer，创建 PeerConnection 并拉起 track fanout。
func (r *Room) Publish(ctx context.Context, offerSDP string) (string, error) {
	r.mu.Lock()
	if r.publisher != nil {
		r.mu.Unlock()
		return "", errors.New("publisher already exists in this room")
	}
	r.mu.Unlock()

	m := &webrtc.MediaEngine{}
	if err := m.PopulateFromSDP(webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: offerSDP}); err != nil {
		return "", fmt.Errorf("populate from SDP: %w", err)
	}
	i := &webrtc.InterceptorRegistry{}
	if err := webrtc.RegisterDefaultInterceptors(m, i); err != nil {
		return "", fmt.Errorf("register interceptors: %w", err)
	}

	api := webrtc.NewAPI(webrtc.WithMediaEngine(m), webrtc.WithInterceptorRegistry(i))
	pc, err := api.NewPeerConnection(r.iceConfig())
	if err != nil {
		return "", err
	}

	pc.OnICEConnectionStateChange(func(s webrtc.ICEConnectionState) {
		if s == webrtc.ICEConnectionStateFailed || s == webrtc.ICEConnectionStateDisconnected || s == webrtc.ICEConnectionStateClosed {
			go r.closePublisher(pc)
		}
	})

	pc.OnTrack(func(remote *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		feed := newTrackFanout(remote, r.name)
		r.mu.Lock()
		r.trackFeeds[remote.ID()] = feed
		// attach existing subscribers
		for sub := range r.subs {
			feed.attachToSubscriber(sub)
		}
		r.mu.Unlock()

		go feed.readLoop()

		go func() {
			// 周期性发送 PLI，提醒发布端刷新关键帧，减轻画面马赛克
			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				r.mu.RLock()
				pub := r.publisher
				r.mu.RUnlock()
				if pub == nil {
					return
				}
				_ = pub.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: uint32(remote.SSRC())}})
			}
		}()

		if r.mgr != nil && r.mgr.cfg != nil && r.mgr.cfg.RecordEnabled {
			// 针对音频/视频分别创建 OGG/IVF 写入器做简单录制
			_ = os.MkdirAll(r.mgr.cfg.RecordDir, 0o755)
			base := fmt.Sprintf("%s_%s_%d", r.name, remote.ID(), time.Now().Unix())
			mime := remote.Codec().MimeType
			switch {
			case mime == webrtc.MimeTypeOpus:
				p := filepath.Join(r.mgr.cfg.RecordDir, base+".ogg")
				if w, err := oggwriter.New(p, 48000, 2); err == nil {
					feed.setRecorder(w, p)
				}
			case mime == webrtc.MimeTypeVP8 || mime == webrtc.MimeTypeVP9:
				p := filepath.Join(r.mgr.cfg.RecordDir, base+".ivf")
				if w, err := ivfwriter.New(p); err == nil {
					feed.setRecorder(w, p)
				}
			}
		}
	})

	if err := pc.SetRemoteDescription(webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: offerSDP}); err != nil {
		_ = pc.Close()
		return "", err
	}
	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		_ = pc.Close()
		return "", err
	}
	g := webrtc.GatheringCompletePromise(pc)
	if err := pc.SetLocalDescription(answer); err != nil {
		_ = pc.Close()
		return "", err
	}
	<-g

	r.mu.Lock()
	r.publisher = pc
	r.mu.Unlock()

	return pc.LocalDescription().SDP, nil
}

// Subscribe 为观众创建 PeerConnection，并把已存在的 track fanout 到新订阅者。
func (r *Room) Subscribe(ctx context.Context, offerSDP string) (string, error) {
	if r.mgr != nil && r.mgr.cfg != nil && r.mgr.cfg.MaxSubsPerRoom > 0 {
		r.mu.RLock()
		if len(r.subs) >= r.mgr.cfg.MaxSubsPerRoom {
			r.mu.RUnlock()
			return "", fmt.Errorf("subscriber limit reached")
		}
		r.mu.RUnlock()
	}
	m := &webrtc.MediaEngine{}
	if err := m.PopulateFromSDP(webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: offerSDP}); err != nil {
		return "", fmt.Errorf("populate from SDP: %w", err)
	}
	i := &webrtc.InterceptorRegistry{}
	if err := webrtc.RegisterDefaultInterceptors(m, i); err != nil {
		return "", fmt.Errorf("register interceptors: %w", err)
	}
	api := webrtc.NewAPI(webrtc.WithMediaEngine(m), webrtc.WithInterceptorRegistry(i))

	pc, err := api.NewPeerConnection(r.iceConfig())
	if err != nil {
		return "", err
	}

	pc.OnICEConnectionStateChange(func(s webrtc.ICEConnectionState) {
		if s == webrtc.ICEConnectionStateFailed || s == webrtc.ICEConnectionStateDisconnected || s == webrtc.ICEConnectionStateClosed {
			go r.removeSubscriber(pc)
		}
	})

	r.mu.RLock()
	for _, feed := range r.trackFeeds {
		feed.attachToSubscriber(pc)
	}
	r.mu.RUnlock()

	if err := pc.SetRemoteDescription(webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: offerSDP}); err != nil {
		_ = pc.Close()
		return "", err
	}

	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		_ = pc.Close()
		return "", err
	}
	g := webrtc.GatheringCompletePromise(pc)
	if err := pc.SetLocalDescription(answer); err != nil {
		_ = pc.Close()
		return "", err
	}
	<-g

	r.mu.Lock()
	r.subs[pc] = struct{}{}
	r.mu.Unlock()
	metrics.IncSubscribers(r.name)

	return pc.LocalDescription().SDP, nil
}

// closePublisher 在发布者掉线时清理资源，并断开所有 fanout。
func (r *Room) closePublisher(pc *webrtc.PeerConnection) {
	r.mu.Lock()
	if r.publisher == pc {
		for _, f := range r.trackFeeds {
			f.close()
		}
		r.trackFeeds = make(map[string]*trackFanout)
		r.publisher = nil
	}
	r.mu.Unlock()
	_ = pc.Close()
}

// removeSubscriber 在订阅者离线时解除与 track fanout 的绑定。
func (r *Room) removeSubscriber(pc *webrtc.PeerConnection) {
	r.mu.Lock()
	if _, ok := r.subs[pc]; ok {
		for _, f := range r.trackFeeds {
			f.detachFromSubscriber(pc)
		}
		delete(r.subs, pc)
	}
	r.mu.Unlock()
	_ = pc.Close()
	metrics.DecSubscribers(r.name)
}

// Close 主动关闭房间内所有连接。
func (r *Room) Close() {
	r.mu.Lock()
	pub := r.publisher
	feeds := r.trackFeeds
	subs := r.subs
	r.publisher = nil
	r.trackFeeds = make(map[string]*trackFanout)
	r.subs = make(map[*webrtc.PeerConnection]struct{})
	r.mu.Unlock()

	if pub != nil {
		_ = pub.Close()
	}
	for _, f := range feeds {
		f.close()
	}
	for s := range subs {
		_ = s.Close()
	}
}

// trackFanout 负责把单个远端 Track 分发给多个订阅者，并可选写盘上传。
type trackFanout struct {
	remote *webrtc.TrackRemote
	mu     sync.RWMutex
	// per-subscriber local tracks
	locals  map[*webrtc.PeerConnection]*webrtc.TrackLocalStaticRTP
	closed  chan struct{}
	room    string
	rec     rtpWriter
	recPath string
}

func newTrackFanout(remote *webrtc.TrackRemote, room string) *trackFanout {
	return &trackFanout{
		remote: remote,
		locals: make(map[*webrtc.PeerConnection]*webrtc.TrackLocalStaticRTP),
		closed: make(chan struct{}),
		room:   room,
	}
}

type rtpWriter interface {
	WriteRTP(*rtp.Packet) error
	Close() error
}

// setRecorder 设置录制写入器与目标文件路径。
func (f *trackFanout) setRecorder(w rtpWriter, path string) {
	f.mu.Lock()
	f.rec = w
	f.recPath = path
	f.mu.Unlock()
}

// attachToSubscriber 为订阅者创建本地 Track，并启动读取循环以清理发送缓冲。
func (f *trackFanout) attachToSubscriber(pc *webrtc.PeerConnection) {
	codec := f.remote.Codec().RTPCodecCapability
	local, err := webrtc.NewTrackLocalStaticRTP(codec, f.remote.ID(), f.remote.StreamID())
	if err != nil {
		return
	}
	sender, err := pc.AddTrack(local)
	if err != nil {
		return
	}
	go func() {
		buf := make([]byte, 1500)
		for {
			if _, _, err := sender.Read(buf); err != nil {
				return
			}
		}
	}()

	f.mu.Lock()
	f.locals[pc] = local
	f.mu.Unlock()
}

func (f *trackFanout) detachFromSubscriber(pc *webrtc.PeerConnection) {
	f.mu.Lock()
	delete(f.locals, pc)
	f.mu.Unlock()
}

// close 关闭录制文件并触发异步上传。
func (f *trackFanout) close() {
	select {
	case <-f.closed:
		return
	default:
		close(f.closed)
	}
	f.mu.Lock()
	if f.rec != nil {
		_ = f.rec.Close()
		if f.recPath != "" {
			go func(p string) { _ = uploader.Upload(context.Background(), p) }(f.recPath)
		}
		f.rec = nil
		f.recPath = ""
	}
	f.mu.Unlock()
}

// readLoop 持续从远端 Track 读取 RTP，并同步写入录制和所有订阅者。
func (f *trackFanout) readLoop() {
	buf := make([]byte, 1500)
	for {
		select {
		case <-f.closed:
			return
		default:
		}
		n, _, err := f.remote.Read(buf)
		if err != nil {
			return
		}
		metrics.AddBytes(f.room, n)
		metrics.IncPackets(f.room)
		pkt := &rtp.Packet{}
		if err := pkt.Unmarshal(buf[:n]); err != nil {
			continue
		}
		f.mu.RLock()
		rec := f.rec
		f.mu.RUnlock()
		if rec != nil {
			_ = rec.WriteRTP(pkt)
		}
		f.mu.RLock()
		for _, local := range f.locals {
			// clone packet for each subscriber to avoid mutation issues
			clone := *pkt
			if pkt.Payload != nil {
				clone.Payload = append([]byte(nil), pkt.Payload...)
			}
			_ = local.WriteRTP(&clone)
		}
		f.mu.RUnlock()
	}
}
