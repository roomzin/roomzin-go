package cluster

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/roomzin/roomzin-go/internal/protocol"
)

type Config struct {
	SeedHosts         string // "host1,host2,host3"  (NO port, NO zone, NO shard)
	APIPort           int    // HTTP port for /peers /leader /node-info
	TCPPort           int    // TCP port for framed protocol
	AuthToken         string
	Timeout           time.Duration
	HttpTimeout       time.Duration
	KeepAlive         time.Duration
	MaxActiveConns    int           // hard cap on open TCP connections
	NodeProbeInterval time.Duration // how often to health-check
}

type Handler struct {
	cfg              *Config
	leaderHandler    *leaderHandler
	followersHandler *followersHandler
	respChanPool     *sync.Pool
	reqPool          *sync.Pool
}

type request struct {
	payload  []byte
	respChan chan protocol.RawResult
	ctx      context.Context
	clrID    uint32
}

type leaderHandler struct {
	cfg     *Config
	reqChan chan *request
	conn    *connection
	connMu  sync.RWMutex
	clrID   uint32
}

type followersHandler struct {
	cfg         *Config
	reqChan     chan *request
	connections map[string]*connection
	connMutex   sync.RWMutex
	clrID       uint32
}

type demuxMap struct {
	mu      sync.RWMutex
	entries map[uint32]demuxEntry
}

type demuxEntry struct {
	ch        chan protocol.RawResult
	send_time time.Time
}

func NewHandler(cfg *Config) *Handler {
	return &Handler{
		cfg: cfg,
		leaderHandler: &leaderHandler{
			cfg:     cfg,
			reqChan: make(chan *request, 1024),
			conn:    nil,
		},
		followersHandler: &followersHandler{
			cfg:         cfg,
			reqChan:     make(chan *request, 1024),
			connections: make(map[string]*connection),
		},
		respChanPool: &sync.Pool{
			New: func() any { return make(chan protocol.RawResult, 1) },
		},
		reqPool: &sync.Pool{
			New: func() any { return &request{} },
		},
	}
}

func (c *Handler) Start(ctx context.Context) {
	go c.leaderHandler.LeaderSyncWorker(ctx)
	go c.followersHandler.FollowerSyncWorker(ctx)

	go c.leaderHandler.LeaderSendWorker(ctx)
	go c.followersHandler.FollowerSendWroker(ctx)

	go func() {
		t := time.NewTicker(c.cfg.Timeout)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				c.leaderHandler.conn.demuxMap.Cleanup(c.cfg.Timeout * 2)
			}
		}
	}()
}

// ========================================================
//   demuxMap – with TTL-based cleanup
// ========================================================

func (m *demuxMap) Store(clrID uint32, ch chan protocol.RawResult) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.entries == nil {
		m.entries = make(map[uint32]demuxEntry)
	}
	m.entries[clrID] = demuxEntry{ch: ch, send_time: time.Now()}
}

func (m *demuxMap) LoadRemove(clrID uint32) (chan protocol.RawResult, time.Time, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.entries[clrID]
	if ok {
		delete(m.entries, clrID)
	}
	return e.ch, e.send_time, ok
}

func (m *demuxMap) Cleanup(maxAge time.Duration) {
	threshold := time.Now().Add(-maxAge)
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, e := range m.entries {
		if e.send_time.Before(threshold) {
			close(e.ch)
			delete(m.entries, id)
		}
	}
}

// ========================================================
//   connection
// ========================================================

type connection struct {
	netConn    net.Conn
	demuxMap   *demuxMap
	sendQueue  chan []byte
	closer     sync.Once
	cfg        *Config
	addr       string
	latency    atomic.Int64    // latest latency
	avgLatency *RollingAverage // moving average of last N samples
	closed     atomic.Bool
}

func newConnection(addr string, cfg *Config, dm *demuxMap) (*connection, error) {
	dialer := &net.Dialer{
		Timeout:   cfg.Timeout,
		KeepAlive: cfg.KeepAlive,
	}
	conn, err := dialer.Dial("tcp", net.JoinHostPort(addr, fmt.Sprintf("%d", cfg.TCPPort)))
	if err != nil {
		return nil, err
	}

	loginPayload, _ := protocol.BuildLoginPayload(cfg.AuthToken)
	frame := protocol.PrependHeader(0, loginPayload)
	if _, err := conn.Write(frame); err != nil {
		conn.Close()
		return nil, err
	}

	const LOGIN_RESP_OK string = "LOGIN OK"
	buf := make([]byte, len(LOGIN_RESP_OK))
	if _, err := conn.Read(buf); err != nil {
		conn.Close()
		return nil, err
	}
	if string(buf) != LOGIN_RESP_OK {
		conn.Close()
		return nil, errors.New("login failed")
	}

	c := &connection{
		netConn:   conn,
		demuxMap:  dm,
		sendQueue: make(chan []byte, 8192),
		cfg:       cfg,
		addr:      addr,
	}
	c.latency.Store(0)
	c.avgLatency = NewRollingAverage(100)

	return c, nil
}

func (c *connection) activate(scored bool) {
	go c.writeLoop()
	go c.readLoop(scored)
}

func (c *connection) writeLoop() {
	// Now start reading from the queue
	for data := range c.sendQueue {
		if _, err := c.netConn.Write(data); err != nil {
			c.Close()
			return
		}
	}
}

// scoring is used for followers
func (c *connection) readLoop(scored bool) {
	for {
		hdr, payload, err := protocol.DrainFrame(c.netConn)
		if err != nil {
			c.Close()
			return
		}

		ch, start, ok := c.demuxMap.LoadRemove(hdr.ClrID)
		if !ok {
			c.Close()
			return
		}

		if scored {
			latency := time.Since(start)
			c.latency.Store(int64(latency))
			c.avgLatency.Add(latency)
		}

		fields, err := protocol.ParseFields(
			payload[1+len(hdr.Status)+2:],
			hdr.FieldCnt,
		)
		if err != nil {
			c.Close()
			return
		}

		// --- handle real-time error hints ---
		if hdr.Status == "ERROR" && len(fields) > 0 {
			switch string(fields[0].Data) {
			case "308": // leader changed
				c.Close() // force leaderHandler sync loop to reconnect

			case "405": // method not allowed - leader rejects reads
				c.Close() // force followerHandler to remove the connection from its list

			case "503": // unavailable
				if scored {
					// penalize latency
					c.avgLatency.Add(c.avgLatency.GetAverage() * 2)
				} else {
					c.Close() // leader unavailable, drop to trigger resync
				}

			case "429": // busy
				if scored {
					// softer penalty
					c.avgLatency.Add(time.Millisecond * 50)
				}
			}
		}

		ch <- protocol.RawResult{Status: hdr.Status, Fields: fields}
	}
}

func (c *connection) Close() error {
	var err error
	c.closer.Do(func() {
		if c.sendQueue != nil {
			close(c.sendQueue)
		}
		if c.netConn != nil {
			err = c.netConn.Close()
		}
		c.closed.Store(true)
	})
	return err
}

func (c *connection) IsClosed() bool {
	return c.closed.Load()
}

// ========================================================
//
//	leaderHandler
//
// ========================================================
func (lh *leaderHandler) updateConnection(newConn *connection) {
	lh.connMu.Lock()
	defer lh.connMu.Unlock()

	// Close old connection if it exists
	if lh.conn != nil {
		lh.conn.Close()
	}

	lh.conn = newConn
}

func (lh *leaderHandler) getConnection() *connection {
	lh.connMu.RLock()
	defer lh.connMu.RUnlock()
	return lh.conn
}

func (lh *leaderHandler) reconnectLeader() {
	curConn := lh.getConnection()

	leaderAddr, _, err := getClusterInfo(lh.cfg)
	if err != nil {
		return
	}

	dm := &demuxMap{}
	if curConn != nil && curConn.demuxMap != nil {
		dm = curConn.demuxMap
	}

	conn, err := newConnection(leaderAddr, lh.cfg, dm)
	if err != nil {
		return
	}

	lh.updateConnection(conn)
	conn.activate(false)

	time.Sleep(10 * time.Millisecond) // Give goroutines time to start
}

func (lh *leaderHandler) LeaderSyncWorker(ctx context.Context) {
	backoff := 100 * time.Millisecond

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		// first check leader is healthy
		curConn := lh.getConnection()
		if curConn == nil || curConn.IsClosed() {
			lh.reconnectLeader()
		}

		// backoff with cap + jitter
		time.Sleep(backoff + time.Duration(rand.Intn(50))*time.Millisecond)
		if backoff < time.Second {
			backoff *= 2
		}
		if backoff > 2*time.Second {
			backoff = 2 * time.Second
		}
	}
}

func (lh *leaderHandler) LeaderSendWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case req := <-lh.reqChan:
			var conn *connection
			// Wait for leader connection to be ready
			for {
				conn = lh.getConnection()
				if conn != nil && conn.netConn != nil && conn.sendQueue != nil {
					break
				}
				select {
				case <-ctx.Done():
					return
				case <-time.After(100 * time.Millisecond):
					// Keep waiting for connection
				}
			}

			clrID := atomic.AddUint32(&lh.clrID, 1)
			conn.demuxMap.Store(clrID, req.respChan)

			frame := protocol.PrependHeader(clrID, req.payload)

			// Try to send with detailed logging
			conn.sendQueue <- frame
		}
	}
}

// ========================================================
//   followersHandler – uses scoring & rebuilds pool
// ========================================================

// RollingAverage tracks a moving average of the last N samples
type RollingAverage struct {
	samples    []atomic.Int64 // Circular buffer of samples
	sum        atomic.Int64   // Current sum of the window
	index      atomic.Int32   // Current position in circular buffer
	count      atomic.Int32   // Current number of samples (until window is full)
	windowSize int32          // Fixed size of the window
}

func NewRollingAverage(windowSize int) *RollingAverage {
	return &RollingAverage{
		samples:    make([]atomic.Int64, windowSize),
		windowSize: int32(windowSize),
	}
}

func (r *RollingAverage) Add(latency time.Duration) {
	idx := r.index.Load()
	oldSample := r.samples[idx].Load()

	// Atomically update the sum: newValue - oldValue + currentSum
	newVal := int64(latency)
	r.sum.Add(newVal - oldSample)

	// Store new value and move index
	r.samples[idx].Store(newVal)
	nextIdx := (idx + 1) % r.windowSize
	r.index.Store(nextIdx)

	// Update count until window is full
	if r.count.Load() < r.windowSize {
		r.count.Add(1)
	}
}

func (r *RollingAverage) GetAverage() time.Duration {
	count := r.count.Load()
	if count == 0 {
		return 0
	}
	sum := r.sum.Load()
	return time.Duration(sum / int64(count))
}

func (fh *followersHandler) getBestConnection() (*connection, error) {
	fh.connMutex.RLock()
	defer fh.connMutex.RUnlock()

	if len(fh.connections) == 0 {
		return nil, errors.New("no follower connections available")
	}

	var best *connection
	var bestAvgLatency time.Duration
	hasLatencyData := false

	// First pass: try to find connection with latency data
	for _, conn := range fh.connections {
		if conn == nil || conn.netConn == nil || conn.sendQueue == nil {
			continue
		}

		avgLatency := conn.avgLatency.GetAverage()
		if avgLatency == 0 {
			continue // Skip if no latency data yet
		}

		hasLatencyData = true
		if best == nil || avgLatency < bestAvgLatency {
			best = conn
			bestAvgLatency = avgLatency
		}
	}

	// If we found connections with latency data, return the best one
	if hasLatencyData {
		return best, nil
	}

	// Second pass: if no latency data yet, return any valid connection
	for _, conn := range fh.connections {
		if conn == nil || conn.netConn == nil || conn.sendQueue == nil {
			continue
		}

		// Return the first valid connection
		return conn, nil
	}

	return nil, errors.New("no valid follower connection found")
}

func (fh *followersHandler) FollowerSendWroker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case req := <-fh.reqChan:
			var conn *connection
			var err error
			for {
				conn, err = fh.getBestConnection()
				if err == nil && conn != nil && conn.netConn != nil && conn.sendQueue != nil {
					break
				}
				select {
				case <-ctx.Done():
					return
				case <-time.After(100 * time.Millisecond):
					// Keep waiting for connection
				}
			}
			clrID := atomic.AddUint32(&fh.clrID, 1)
			conn.demuxMap.Store(clrID, req.respChan)
			frame := protocol.PrependHeader(clrID, req.payload)
			conn.sendQueue <- frame
		}
	}
}

func (fh *followersHandler) reconnectFollower(addr string) {
	fh.connMutex.Lock()
	defer fh.connMutex.Unlock()

	if con, ok := fh.connections[addr]; ok && con != nil && !con.IsClosed() {
		return // already connected
	}

	conn, err := newConnection(addr, fh.cfg, &demuxMap{})
	if err != nil {
		return
	}
	if conn.netConn == nil {
		return
	}
	if conn.sendQueue == nil {
		return
	}

	fh.connections[addr] = conn
	conn.activate(true)
}

func (fh *followersHandler) syncFollowers() {
	_, followers, err := getClusterInfo(fh.cfg)
	if err != nil {
		return
	}

	// --- add new followers we do not have yet ---
	for _, a := range followers {
		fh.reconnectFollower(a)
	}
}

func (fh *followersHandler) FollowerSyncWorker(ctx context.Context) {
	ticker := time.NewTicker(fh.cfg.NodeProbeInterval)
	fastTick := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	defer fastTick.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-fastTick.C:
			active := len(fh.connections)
			for _, con := range fh.connections {
				if con == nil || con.IsClosed() {
					active -= 1
				}
			}
			if active == 0 {
				fh.syncFollowers()
			}
		case <-ticker.C:
			fh.syncFollowers()
		}
	}
}

// clrID is reset on every retry; uses config timeouts.
func (c *Handler) Execute(ctx context.Context, isWrite bool, payload []byte) (protocol.RawResult, error) {
	if len(payload) == 0 {
		return protocol.RawResult{}, errors.New("payload should not be empty")
	}

	reqAny := c.reqPool.Get()
	if reqAny == nil {
		return protocol.RawResult{}, errors.New("failed to get request from pool")
	}
	req := reqAny.(*request)
	defer c.reqPool.Put(req)

	respAny := c.respChanPool.Get()
	if respAny == nil {
		return protocol.RawResult{}, errors.New("failed to get respChan from pool")
	}
	respChan := respAny.(chan protocol.RawResult)
	defer c.respChanPool.Put(respChan)

	req.payload = payload
	req.ctx = ctx
	req.respChan = respChan
	req.clrID = 0 // will be set on send

	// choose handler
	handlerChan := c.followersHandler.reqChan
	if isWrite {
		handlerChan = c.leaderHandler.reqChan
	}

	// retry policy
	maxRetries := 5
	attempts := 0

	send := func() error {
		select {
		case handlerChan <- req:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// send first attempt
	if err := send(); err != nil {
		return protocol.RawResult{}, err
	}

	for {
		select {
		case <-ctx.Done():
			return protocol.RawResult{}, ctx.Err()

		case res := <-respChan:
			if res.Status == "SUCCESS" {
				return res, nil
			}

			// res.Status == "ERROR"
			var errMsg string
			if len(res.Fields) > 0 {
				errMsg = string(res.Fields[0].Data)
			} else {
				errMsg = res.Status
			}
			backoff := false
			switch errMsg {
			case "405", "308":
				// 405: follower node is promoted to leader and rejects reads
				// 308: leader changed
			case "503", "429": // unavailable / busy
				backoff = true
			default:
				return res, errors.New("max retries reached after " + errMsg)
			}

			if attempts >= maxRetries {
				return res, errors.New("max retries reached after " + errMsg)
			}
			attempts++

			// first retry immediately, then backoff
			if backoff {
				select {
				case <-time.After(time.Duration(attempts) * 100 * time.Millisecond):
				case <-ctx.Done():
					return protocol.RawResult{}, ctx.Err()
				}
			}

			req.clrID = 0
			if err := send(); err != nil {
				return res, errors.New("retry failed after " + errMsg + ": " + err.Error())
			}
		}
	}
}
