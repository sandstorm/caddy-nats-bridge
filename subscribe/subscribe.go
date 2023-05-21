package subscribe

import (
	"bytes"
	"fmt"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/nats-io/nats.go"
	"github.com/sandstorm/caddy-nats-bridge/common"
	"go.uber.org/zap"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
)

type Subscribe struct {
	Subject    string `json:"subject,omitempty"`
	Method     string `json:"method,omitempty"`
	URL        string `json:"path,omitempty"`
	QueueGroup string `json:"queue_group,omitempty"`

	conn    *nats.Conn
	sub     *nats.Subscription
	ctx     caddy.Context
	logger  *zap.Logger
	httpApp *caddyhttp.App
}

func (Subscribe) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "nats.handlers.subscribe",
		New: func() caddy.Module { return new(Subscribe) },
	}
}

func (s *Subscribe) Provision(ctx caddy.Context) error {
	s.ctx = ctx
	s.logger = ctx.Logger()

	return nil
}

func (s *Subscribe) Subscribe(conn *nats.Conn) error {
	s.logger.Info(
		"subscribing to NATS subject",
		zap.String("subject", s.Subject),
		zap.String("queue_group", s.QueueGroup),
		zap.String("method", s.Method),
		zap.String("url", s.URL),
	)

	httpAppIface, err := s.ctx.App("http")
	if err != nil {
		return err
	}
	s.httpApp = httpAppIface.(*caddyhttp.App)
	s.conn = conn

	if s.QueueGroup != "" {
		s.sub, err = conn.QueueSubscribe(s.Subject, s.QueueGroup, s.handler)
	} else {
		s.sub, err = conn.Subscribe(s.Subject, s.handler)
	}

	return err
}

func (s *Subscribe) Unsubscribe(conn *nats.Conn) error {
	s.logger.Info(
		"unsubscribing from NATS subject",
		zap.String("subject", s.Subject),
		zap.String("queue_group", s.QueueGroup),
		zap.String("method", s.Method),
		zap.String("url", s.URL),
	)

	return s.sub.Drain()
}

func (s *Subscribe) handler(msg *nats.Msg) {
	repl := caddy.NewReplacer()
	common.AddNatsSubscribeVarsToReplacer(repl, msg)

	url := repl.ReplaceAll(s.URL, "")
	method := repl.ReplaceAll(s.Method, "")

	s.logger.Debug(
		"handling message NATS on subject",
		zap.String("subject", msg.Subject),
		zap.String("queue_group", s.QueueGroup),
		zap.String("method", method),
		zap.String("url", url),
		zap.Bool("with_reply", msg.Reply != ""),
	)

	req, err := s.prepareRequest(method, url, bytes.NewBuffer(msg.Data), msg.Header)
	if err != nil {
		s.logger.Error("error creating request", zap.Error(err))
		return
	}

	server, err := s.matchServer(s.httpApp.Servers, req)
	if err != nil {
		s.logger.Error("error matching server", zap.Error(err))
		return
	}

	if msg.Reply != "" {
		// the incoming NATS Message has a reply subject set; so it was sent via request() (and not via publish()).
		// -> so we can send the response back.
		rec := httptest.NewRecorder()
		server.ServeHTTP(rec, req)
		// rec.Code -> TODO: new status code
		//TODO Handle error
		msg.RespondMsg(&nats.Msg{
			Header: nats.Header(rec.Header()),
			Data:   rec.Body.Bytes(),
		})
		return
	}

	// no reply subject was set -> the original NATS requester is not interested in the response - we can ignore it.
	server.ServeHTTP(common.NoopResponseWriter{}, req)
}

func (s *Subscribe) matchServer(servers map[string]*caddyhttp.Server, req *http.Request) (*caddyhttp.Server, error) {
	repl := caddy.NewReplacer()
	for _, server := range servers {
		if !hasListenerAddress(server, req.URL) {
			// listener address (host/port) does not match for server, so we can continue with next server.
			continue
		}
		r := caddyhttp.PrepareRequest(req, repl, nil, server)
		for _, route := range server.Routes {

			if route.MatcherSets.AnyMatch(r) {
				return server, nil
			}
		}
	}

	return nil, fmt.Errorf("no server matched for the current url: %s", req.URL)
}

// hasListenerAddress is modelled after the same function in caddyhttp/server, but a bit simplified.
func hasListenerAddress(s *caddyhttp.Server, fullAddr *url.URL) bool {
	laddrs, err := caddy.ParseNetworkAddress(fullAddr.Host)
	if err != nil {
		return false
	}
	if laddrs.PortRangeSize() != 1 {
		return false // TODO: support port ranges
	}

	for _, lnAddr := range s.Listen {
		thisAddrs, err := caddy.ParseNetworkAddress(lnAddr)
		if err != nil {
			continue
		}
		if thisAddrs.Network != laddrs.Network {
			continue
		}

		if (laddrs.StartPort <= thisAddrs.EndPort) &&
			(laddrs.StartPort >= thisAddrs.StartPort) {
			return true
		}
	}
	return false
}

func (s *Subscribe) prepareRequest(method string, rawURL string, body io.Reader, header nats.Header) (*http.Request, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid url: %s", rawURL)
	}

	req, err := http.NewRequest(method, rawURL, body)
	if header != nil {
		req.Header = http.Header(header)
	}

	req.RequestURI = u.Path
	req.RemoteAddr = s.conn.ConnectedAddr()
	//TODO: make User-Agent configurable
	req.Header.Add("User-Agent", "caddy-nats")

	return req, err
}

var (
	_ caddy.Provisioner  = (*Subscribe)(nil)
	_ common.NatsHandler = (*Subscribe)(nil)
)
