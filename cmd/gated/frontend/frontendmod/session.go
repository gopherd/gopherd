package frontendmod

import (
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gopherd/doge/net/netutil"
	"github.com/gopherd/doge/proto"
	"github.com/gopherd/doge/text/resp"
	"github.com/gopherd/gopherd/proto/gatepb"
	"github.com/gopherd/jwt"
	"github.com/gopherd/log"
)

// session state enumerator
type state int

const (
	stateCreated state = iota
	statePendingLogin
	stateLogged
	stateOverflow
	stateClosing
	stateClosed
)

const (
	maxDurationForPendingSession = time.Second * 5
	handlePendingSessionInterval = time.Millisecond * 200
	cleanDeadSessionInterval     = time.Minute
	userInfoTTLRatio             = 750 // 750/1000
)

// userdata of session
type user struct {
	token jwt.Payload
}

// session event handler
type handler interface {
	onReady(*session)
	onClose(*session, error)
	onMessage(*session, proto.Type, proto.Body) error
	onCommand(*session, *resp.Command) error
}

// session holds a context for each connection
type session struct {
	id        int64
	ip        string
	createdAt int64
	handler   handler
	logger    log.Prefix

	// private fields of the session
	internal struct {
		session           *netutil.Session
		state             int32
		uid               int64
		user              user
		lastKeepaliveTime int64
		lastUpdateSidTime int64
		currSceneId       int64
		stats             struct {
			recv int64
			send int64
		}
		writerMu sync.Mutex
	}
}

func newSession(prefix log.Prefix, id int64, ip string, conn net.Conn, handler handler) *session {
	s := &session{
		id:        id,
		ip:        ip,
		createdAt: time.Now().UnixNano() / 1e6,
		handler:   handler,
		logger:    prefix.Prefix("sid/" + strconv.FormatInt(id, 10)),
	}
	s.internal.state = int32(stateCreated)
	s.internal.session = netutil.NewSession(conn, s)
	return s
}

func (s *session) keepalive() {
	atomic.StoreInt64(&s.internal.lastKeepaliveTime, time.Now().UnixNano()/1e6)
}

func (s *session) ContentType() proto.ContentType {
	return s.internal.session.ContentType()
}

// OnOpen implements netutil.SessionEventHandler OnOpen method
func (s *session) OnOpen() {
	s.logger.Trace().Print("session open")
	s.keepalive()
	s.handler.onReady(s)
}

// OnClose implements netutil.SessionEventHandler OnClose method
func (s *session) OnClose(err error) {
	s.setState(stateClosed)
	if !netutil.IsNetworkError(err) {
		s.logger.Warn().
			Error("error", err).
			Print("session closed because of an error occurred")
	} else {
		s.logger.Debug().Print("session closed")
	}
	s.handler.onClose(s, err)
}

// OnHandshake implements netutil.SessionEventHandler OnHandshake method
func (s *session) OnHandshake(contentType proto.ContentType) error {
	switch contentType {
	case proto.ContentTypeText, proto.ContentTypeProtobuf:
		return nil
	default:
		return proto.ErrUnsupportedContentType
	}
}

// OnMessage implements netutil.SessionEventHandler OnMessage method
func (s *session) OnMessage(typ proto.Type, body proto.Body) error {
	atomic.AddInt64(&s.internal.stats.recv, int64(body.Len()))
	s.keepalive()
	return s.handler.onMessage(s, typ, body)
}

// OnCommand implements netutil.CommandHandler OnCommand method
func (s *session) OnCommand(cmd *resp.Command) error {
	s.keepalive()
	return s.handler.onCommand(s, cmd)
}

// serve runs the session read/write loops
func (s *session) serve() {
	s.internal.session.Serve()
}

// Write writes data to underlying connection
func (s *session) Write(data []byte) (int, error) {
	atomic.AddInt64(&s.internal.stats.send, int64(len(data)))
	s.internal.writerMu.Lock()
	defer s.internal.writerMu.Unlock()
	return s.internal.session.Write(data)
}

// Close closes the session
func (s *session) Close(err error) {
	s.setState(stateClosing)
	s.logger.Debug().Error("error", err).Print("close session")
	s.internal.session.Close(err)
}

func (s *session) send(m proto.Message) error {
	buf := proto.AllocBuffer()
	defer proto.FreeBuffer(buf)
	if err := buf.Encode(m, s.internal.session.ContentType()); err != nil {
		return err
	}
	_, err := s.Write(buf.Bytes())
	return err
}

func (s *session) getState() state {
	return state(atomic.LoadInt32(&s.internal.state))
}

func (s *session) setState(state state) {
	atomic.StoreInt32(&s.internal.state, int32(state))
}

func (s *session) getUid() int64 {
	return atomic.LoadInt64(&s.internal.uid)
}

func (s *session) getUser() user {
	return s.internal.user
}

func (s *session) setUser(user user) {
	s.internal.user = user
	atomic.StoreInt64(&s.internal.uid, user.token.ID)
}

func (s *session) getLastKeepaliveTime() int64 {
	return atomic.LoadInt64(&s.internal.lastKeepaliveTime)
}

func (s *session) trySetLastUpdateSidTime(ttl, now int64) bool {
	old := atomic.LoadInt64(&s.internal.lastUpdateSidTime)
	if now > old+ttl {
		return atomic.CompareAndSwapInt64(&s.internal.lastUpdateSidTime, old, now)
	}
	return false
}

type pendingSession struct {
	uid  int64
	meta uint32
}

const (
	maxbuckets      = 8
	maxsizeofbucket = 1024
)

type sessions struct {
	mod *frontendModule

	maxConns      int
	maxConnsPerIP int
	nextSessionId int64
	nextbucket    int

	mutex   sync.RWMutex
	uid2sid map[int64]int64
	ips     map[string]int
	nbucket int
	buckets [maxbuckets]map[int64]*session
}

func newSessions(mod *frontendModule) *sessions {
	ss := &sessions{
		mod:     mod,
		uid2sid: make(map[int64]int64),
		ips:     make(map[string]int),
		nbucket: 1,
	}
	for i := 0; i < ss.nbucket; i++ {
		ss.buckets[i] = make(map[int64]*session)
	}
	return ss
}

func (ss *sessions) init() {
	cfg := ss.mod.service.Config()
	ss.maxConns = cfg.MaxConns
	ss.maxConnsPerIP = cfg.MaxConnsPerIP
}

func (ss *sessions) allocSessionId() int64 {
	return atomic.AddInt64(&ss.nextSessionId, 1)
}

func (ss *sessions) expand() {
	if ss.nbucket >= maxbuckets {
		return
	}
	old := ss.nbucket
	ss.nbucket *= 2
	mask := ss.nbucket - 1
	for i := old; i < ss.nbucket; i++ {
		ss.buckets[i] = make(map[int64]*session)
		b := ss.buckets[i-old]
		for id, s := range b {
			if id&int64(mask) == int64(i) {
				delete(b, id)
				ss.buckets[i][id] = s
			}
		}
	}
}

func (ss *sessions) shutdown() {
	ss.mutex.RLock()
	defer ss.mutex.RUnlock()
	for i := 0; i < ss.nbucket; i++ {
		b := ss.buckets[i]
		for _, s := range b {
			if state := s.getState(); state == stateClosing || state == stateOverflow {
				continue
			}
			s.send(&gatepb.Kickout{
				Reason: int32(gatepb.KickoutReason_ReasonServiceClosed),
			})
			s.Close(nil)
		}
	}
}

func (ss *sessions) ttl() int64 {
	return 2 * int64(ss.mod.service.Config().Keepalive) * 1000
}

func (ss *sessions) clean(now time.Time) {
	ss.mutex.RLock()
	defer ss.mutex.RUnlock()
	if ss.nextbucket >= ss.nbucket {
		ss.nextbucket = 0
	}
	ttl := ss.ttl()
	timestamp := now.UnixNano() / 1e6
	for id, s := range ss.buckets[ss.nextbucket] {
		last := s.getLastKeepaliveTime()
		if timestamp >= last+ttl {
			ss.mod.Logger().Debug().
				Int64("sid", id).
				Print("close inactive session")
			s.Close(nil)
		}
	}
	ss.nextbucket++
}

func (ss *sessions) size() int {
	ss.mutex.RLock()
	defer ss.mutex.RUnlock()
	size := 0
	for i := 0; i < ss.nbucket; i++ {
		size += len(ss.buckets[i])
	}
	return size
}

func (ss *sessions) add(s *session) (n int, ok bool) {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()

	i := s.id & int64(ss.nbucket-1)
	b := ss.buckets[i]
	if len(b) >= maxsizeofbucket {
		ss.expand()
		i = s.id & int64(ss.nbucket-1)
		b = ss.buckets[i]
	}
	b[s.id] = s
	for i := 0; i < ss.nbucket; i++ {
		n += len(ss.buckets[i])
	}
	if ss.maxConns == 0 || n < ss.maxConns {
		ok = true
	} else {
		s.setState(stateOverflow)
	}
	return
}

func (ss *sessions) remove(id int64) *session {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()

	i := id & int64(ss.nbucket-1)
	b := ss.buckets[i]
	s, ok := b[id]
	if !ok {
		return nil
	}
	delete(b, id)

	ip := s.ip
	if n, ok := ss.ips[ip]; n > 1 {
		ss.ips[ip] = n - 1
	} else if ok {
		delete(ss.ips, ip)
	}
	if uid := s.getUid(); uid > 0 {
		delete(ss.uid2sid, uid)
	}
	return s
}

func (ss *sessions) mapping(uid, sid int64) bool {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()
	if old, ok := ss.uid2sid[uid]; ok {
		if sid != old {
			ok = false
		}
		return ok
	}
	ss.uid2sid[uid] = sid
	return true
}

func (ss *sessions) get(sid int64) *session {
	ss.mutex.RLock()
	defer ss.mutex.RUnlock()
	b := ss.buckets[sid&int64(ss.nbucket-1)]
	return b[sid]
}

func (ss *sessions) find(uid int64) *session {
	ss.mutex.RLock()
	defer ss.mutex.RUnlock()
	sid, ok := ss.uid2sid[uid]
	if !ok {
		return nil
	}
	b := ss.buckets[sid&int64(ss.nbucket-1)]
	return b[sid]
}

func (ss *sessions) recordIP(sid int64, ip string) bool {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()
	if n := ss.ips[ip]; n < ss.maxConnsPerIP {
		ss.ips[ip] = n + 1
		return true
	}
	return false
}

func (ss *sessions) broadcast(data []byte, now int64) {
	ss.mutex.RLock()
	defer ss.mutex.Unlock()
	ttl := ss.ttl()
	for _, b := range ss.buckets {
		for sid, s := range b {
			if s.getLastKeepaliveTime()+ttl < now {
				ss.mod.Logger().Debug().
					Int64("sid", sid).
					Print("close inactive session")
				s.Close(nil)
				continue
			}
			if _, err := s.Write(data); err != nil {
				ss.mod.Logger().Warn().
					Int64("sid", sid).
					Int("size", len(data)).
					Error("error", err).
					Print("broadcast to session error")
			}
		}
	}
}
