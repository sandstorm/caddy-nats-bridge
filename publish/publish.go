package publish

import (
	"fmt"
	"net/http"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/sandstorm/caddy-nats-bridge/common"
	"github.com/sandstorm/caddy-nats-bridge/natsbridge"
	"go.uber.org/zap"
)

type Publish struct {
	Subject     string `json:"subject,omitempty"`
	ServerAlias string `json:"serverAlias,omitempty"`
	Headers     bool   `json:"headers,omitempty"`

	logger *zap.Logger
	app    *natsbridge.NatsBridgeApp
}

func (Publish) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "http.handlers.nats_publish",
		New: func() caddy.Module {
			// Default values
			return &Publish{
				ServerAlias: "default",
				Headers:     true,
			}
		},
	}
}

func (p *Publish) Provision(ctx caddy.Context) error {
	p.logger = ctx.Logger(p)

	natsAppIface, err := ctx.App("nats")
	if err != nil {
		return fmt.Errorf("getting NATS app: %v. Make sure NATS is configured in nats options", err)
	}

	p.app = natsAppIface.(*natsbridge.NatsBridgeApp)

	return nil
}

func (p Publish) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	repl := r.Context().Value(caddy.ReplacerCtxKey).(*caddy.Replacer)
	common.AddNATSPublishVarsToReplacer(repl, r)

	//TODO: What method is best here? ReplaceAll vs ReplaceWithErr?
	subj := repl.ReplaceAll(p.Subject, "")

	p.logger.Debug("publishing NATS message", zap.String("subject", subj))

	server, ok := p.app.Servers[p.ServerAlias]
	if !ok {
		return fmt.Errorf("NATS server alias %s not found", p.ServerAlias)
	}

	msg, err := common.NatsMsgForHttpRequest(r, subj, p.Headers)
	if err != nil {
		return err
	}

	err = server.Conn.PublishMsg(msg)
	if err != nil {
		return fmt.Errorf("could not publish NATS message: %w", err)
	}

	// TODO: wiretap mode :) -> Response to NATS.
	return next.ServeHTTP(w, r)
}

var (
	_ caddyhttp.MiddlewareHandler = (*Publish)(nil)
	_ caddy.Provisioner           = (*Publish)(nil)
	_ caddyfile.Unmarshaler       = (*Publish)(nil)
)
