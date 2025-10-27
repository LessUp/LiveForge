package sfu

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
	"os"
	"path/filepath"

	"live-webrtc-go/internal/config"
	"live-webrtc-go/internal/metrics"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media/ivfwriter"
	"github.com/pion/webrtc/v3/pkg/media/oggwriter"
)

type Manager struct {
	mu    sync.RWMutex
	rooms map[string]*Room
	cfg   *config.Config
}

func NewManager(c *config.Config) *Manager {
	return &Manager{rooms: make(map[string]*Room), cfg: c}
}

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

func (m *Manager) Publish(ctx context.Context, roomName, offerSDP string) (string, error) {
	r := m.getOrCreateRoom(roomName)
	return r.Publish(ctx, offerSDP)
}

func (m *Manager) Subscribe(ctx context.Context, roomName, offerSDP string) (string, error) {
	r := m.getOrCreateRoom(roomName)
	return r.Subscribe(ctx, offerSDP)
}

type RoomInfo struct {
	Name          string
	HasPublisher  bool
	Tracks        int
	Subscribers   int
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

type Room struct {
	name       string
	mu         sync.RWMutex
	publisher  *webrtc.PeerConnection
	trackFeeds map[string]*trackFanout // key: track ID
	subs       map[*webrtc.PeerConnection]struct{}
	mgr        *Manager
}

func NewRoom(name string, m *Manager) *Room {
	return &Room{
		name:       name,
		trackFeeds: make(map[string]*trackFanout),
		subs:       make(map[*webrtc.PeerConnection]struct{}),
		mgr:        m,
	}
}

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
			_ = os.MkdirAll(r.mgr.cfg.RecordDir, 0o755)
			base := fmt.Sprintf("%s_%s_%d", r.name, remote.ID(), time.Now().Unix())
			mime := remote.Codec().MimeType
			switch {
			case mime == webrtc.MimeTypeOpus:
				p := filepath.Join(r.mgr.cfg.RecordDir, base+".ogg")
				if w, err := oggwriter.New(p, 48000, 2); err == nil {
					feed.setRecorder(w)
				}
			case mime == webrtc.MimeTypeVP8 || mime == webrtc.MimeTypeVP9:
				p := filepath.Join(r.mgr.cfg.RecordDir, base+".ivf")
				if w, err := ivfwriter.New(p); err == nil {
					feed.setRecorder(w)
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

type trackFanout struct {
	remote *webrtc.TrackRemote
	mu     sync.RWMutex
	// per-subscriber local tracks
	locals map[*webrtc.PeerConnection]*webrtc.TrackLocalStaticRTP
	closed chan struct{}
	room   string
	rec    rtpWriter
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

func (f *trackFanout) setRecorder(w rtpWriter) {
	f.mu.Lock()
	f.rec = w
	f.mu.Unlock()
}

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
		f.rec = nil
	}
	f.mu.Unlock()
}

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
