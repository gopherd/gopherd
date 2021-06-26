package internal

import (
	"net"
	"net/textproto"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/gopherd/doge/net/netutil"
	"github.com/gopherd/doge/proto"
	"github.com/gopherd/jwt"
	"github.com/gopherd/log"
)

// userdata of session
type user struct {
	device string
	token  jwt.Payload
}

// session state enumerator
type state int

const (
	stateCreated state = iota
	statePendingLogin
	stateLogged
	stateClosing
	stateOverflow
)

// session event handler
type handler interface {
	onReady(*session)
	onClose(*session, error)
	onMessage(*session, proto.Type, proto.Body) error
	onTextMessage(*session, *textproto.Reader) error
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
		// (TODO): limiter
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

// OnReady implements netutil.SessionEventHandler OnReady method
func (s *session) OnReady() {
	s.logger.Trace().Print("session ready")
	s.keepalive()
	s.handler.onReady(s)
}

// OnClose implements netutil.SessionEventHandler OnClose method
func (s *session) OnClose(err error) {
	if !netutil.IsNetworkError(err) {
		s.logger.Warn().
			Error("error", err).
			Print("session closed because of an error occurred")
	} else {
		s.logger.Debug().Print("session closed")
	}
	s.handler.onClose(s, err)
}

// OnMessage implements netutil.SessionEventHandler OnMessage method
func (s *session) OnMessage(typ proto.Type, body proto.Body) error {
	atomic.AddInt64(&s.internal.stats.recv, int64(body.Len()))
	s.keepalive()
	return s.handler.onMessage(s, typ, body)
}

// OnMessage implements netutil.TextMessageHandler OnTextMessage method
func (s *session) OnTextMessage(body *textproto.Reader) error {
	return s.handler.onTextMessage(s, body)
}

// serve runs the session read/write loops
func (s *session) serve() {
	s.internal.session.Serve()
}

// Write writes data to underlying connection
func (s *session) Write(data []byte) (int, error) {
	atomic.AddInt64(&s.internal.stats.send, int64(len(data)))
	return s.internal.session.Write(data)
}

// Close closes the session
func (s *session) Close() error {
	s.logger.Debug().Print("close session")
	s.setState(stateClosing)
	return s.internal.session.Close()
}

func (sess *session) send(m proto.Message) error {
	buf := proto.AllocBuffer()
	defer proto.FreeBuffer(buf)
	if err := buf.Encode(m); err != nil {
		return err
	}
	_, err := sess.Write(buf.Bytes())
	return err
}

func (sess *session) sendTextResponse(text string) error {
	_, err := sess.Write(proto.TextprotoPrefix())
	if err != nil {
		return err
	}
	_, err = sess.Write([]byte(text))
	if err != nil {
		return err
	}
	_, err = sess.Write(proto.TextprotoSuffix())
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

func (s *session) setUser(user user) {
	s.internal.user = user
	atomic.StoreInt64(&s.internal.uid, user.token.Uid)
}

func (s *session) getLastKeepaliveTime() int64 {
	return atomic.LoadInt64(&s.internal.lastKeepaliveTime)
}

type pendingSession struct {
	uid  int64
	meta uint32
}
