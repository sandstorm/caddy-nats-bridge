package request

import (
	"fmt"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
	"io"
	"net/http"
	"sandstorm.de/custom-caddy/nats-bridge/common"
	"sandstorm.de/custom-caddy/nats-bridge/global"
	"time"
)

const publishDefaultTimeout = 10000

func init() {

}

type Request struct {
	Subject     string        `json:"subject,omitempty"`
	Timeout     time.Duration `json:"timeout,omitempty"`
	ServerAlias string        `json:"serverAlias,omitempty"`

	logger *zap.Logger
	app    *global.NatsBridgeApp
}

func (Request) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.nats_request",
		New: func() caddy.Module { return new(Request) },
	}
}

func (p *Request) Provision(ctx caddy.Context) error {
	p.logger = ctx.Logger(p)

	natsAppIface, err := ctx.App("nats")
	if err != nil {
		return fmt.Errorf("getting NATS app: %v. Make sure NATS is configured in global options", err)
	}

	p.app = natsAppIface.(*global.NatsBridgeApp)

	return nil
}

func (p Request) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	repl := r.Context().Value(caddy.ReplacerCtxKey).(*caddy.Replacer)
	//addNATSPublishVarsToReplacer(repl, r)

	//TODO: What method is best here? ReplaceAll vs ReplaceWithErr?
	subj := repl.ReplaceAll(p.Subject, "")

	//TODO: Check max msg size

	//p.logger.Debug("publishing NATS message", zap.String("subject", subj), zap.Bool("with_reply", p.WithReply), zap.Int64("timeout", p.Timeout))
	p.logger.Debug("publishing NATS message", zap.String("subject", subj))

	/*if p.WithReply {
		return p.natsRequestReply(subj, r, w)
	}*/

	server, ok := p.app.Servers[p.ServerAlias]
	if !ok {
		return fmt.Errorf("NATS server alias %s not found", p.ServerAlias)
	}

	msg, err := p.natsMsgForHttpRequest(r, subj, server)
	if err != nil {
		return err
	}

	err = server.Conn.PublishMsg(msg)
	if err != nil {
		return fmt.Errorf("could not publish NATS message: %w", err)
	}
	return next.ServeHTTP(w, r)
}

func (p *Request) natsMsgForHttpRequest(r *http.Request, subject string, server *global.NatsServer) (*nats.Msg, error) {
	var msg *nats.Msg
	// TODO: real message size limit of NATS here

	// NOTE: we could implement this in a streaming fashion to JetStream via
	// _, err = os.Put(&nats.ObjectMeta{
	//			Name: fileStreamId,
	//		}, r.Body)
	// but we could not make this work.
	// So that's why we can easily read the full body here anyways to simplify code paths; and then we can
	// decide based on the actual length; and not based of the ContentLength Header.
	// In case we want to change it somewhen, we need to take care of Chunked Uploads via r.ContentLength == -1 || r.ContentLength > 950_000_000
	b, _ := io.ReadAll(r.Body)

	headers := nats.Header(r.Header)
	for k, v := range common.ExtraNatsMsgHeadersFromContext(r.Context()) {
		headers.Add(k, v)
	}
	msg = &nats.Msg{
		Subject: subject,
		Header:  headers,
		Data:    b,
	}

	msg.Header.Add("X-NatsHttp-Method", r.Method)
	msg.Header.Add("X-NatsHttp-UrlPath", r.URL.Path)
	msg.Header.Add("X-NatsHttp-UrlQuery", r.URL.RawQuery)
	return msg, nil
}

//
//func (p *Publish) natsRequestReply(subject string, r *http.Request, w http.ResponseWriter) error {
//	msg, err := p.natsMsgForHttpRequest(r, subject)
//	if err != nil {
//		return err
//	}
//	m, err := p.app.conn.RequestMsg(msg, time.Duration(p.Timeout)*time.Millisecond)
//
//	// nats.ErrMaxPayload
//
//	//data, err := io.ReadAll(r.Body)
//	//if err != nil {
//	//	return err
//	//}
//
//	//os.AddLink()
//	// TODO: Make error handlers configurable
//	if err == nats.ErrNoResponders {
//		w.WriteHeader(http.StatusNotFound)
//		return err
//	} else if err != nil {
//		w.WriteHeader(http.StatusInternalServerError)
//		return err
//	}
//
//	_, err = w.Write(m.Data)
//
//	return err
//}

var (
	_ caddyhttp.MiddlewareHandler = (*Request)(nil)
	_ caddy.Provisioner           = (*Request)(nil)
	_ caddyfile.Unmarshaler       = (*Request)(nil)
)
