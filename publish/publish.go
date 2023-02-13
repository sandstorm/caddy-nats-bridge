package publish

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
)

type Publish struct {
	Subject     string `json:"subject,omitempty"`
	ServerAlias string `json:"serverAlias,omitempty"`

	logger *zap.Logger
	app    *global.NatsBridgeApp
}

func (Publish) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.nats_publish",
		New: func() caddy.Module { return new(Publish) },
	}
}

func (p *Publish) Provision(ctx caddy.Context) error {
	p.logger = ctx.Logger(p)

	natsAppIface, err := ctx.App("nats")
	if err != nil {
		return fmt.Errorf("getting NATS app: %v. Make sure NATS is configured in global options", err)
	}

	p.app = natsAppIface.(*global.NatsBridgeApp)

	return nil
}

func (p Publish) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	repl := r.Context().Value(caddy.ReplacerCtxKey).(*caddy.Replacer)
	//addNATSPublishVarsToReplacer(repl, r)

	//TODO: What method is best here? ReplaceAll vs ReplaceWithErr?
	subj := repl.ReplaceAll(p.Subject, "")

	p.logger.Debug("publishing NATS message", zap.String("subject", subj))

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

func (p *Publish) natsMsgForHttpRequest(r *http.Request, subject string, server *global.NatsServer) (*nats.Msg, error) {
	var msg *nats.Msg

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

var (
	_ caddyhttp.MiddlewareHandler = (*Publish)(nil)
	_ caddy.Provisioner           = (*Publish)(nil)
	_ caddyfile.Unmarshaler       = (*Publish)(nil)
)
