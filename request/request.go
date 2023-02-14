package request

import (
	"fmt"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/nats-io/nats.go"
	"github.com/sandstorm/caddy-nats-bridge/common"
	"github.com/sandstorm/caddy-nats-bridge/natsbridge"
	"go.uber.org/zap"
	"net/http"
	"time"
)

type Request struct {
	Subject     string        `json:"subject,omitempty"`
	Timeout     time.Duration `json:"timeout,omitempty"`
	ServerAlias string        `json:"serverAlias,omitempty"`

	logger *zap.Logger
	app    *natsbridge.NatsBridgeApp
}

func (Request) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "http.handlers.nats_request",
		New: func() caddy.Module {
			// Default values
			return &Request{
				Timeout:     1 * time.Second,
				ServerAlias: "default",
			}
		},
	}
}

func (p *Request) Provision(ctx caddy.Context) error {
	p.logger = ctx.Logger(p)

	natsAppIface, err := ctx.App("nats")
	if err != nil {
		return fmt.Errorf("getting NATS app: %v. Make sure NATS is configured in nats options", err)
	}

	p.app = natsAppIface.(*natsbridge.NatsBridgeApp)

	return nil
}

func (p Request) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	repl := r.Context().Value(caddy.ReplacerCtxKey).(*caddy.Replacer)
	common.AddNATSPublishVarsToReplacer(repl, r)

	//TODO: What method is best here? ReplaceAll vs ReplaceWithErr?
	subj := repl.ReplaceAll(p.Subject, "")

	//p.logger.Debug("publishing NATS message", zap.String("subject", subj), zap.Bool("with_reply", p.WithReply), zap.Int64("timeout", p.Timeout))
	p.logger.Debug("publishing NATS message", zap.String("subject", subj))

	server, ok := p.app.Servers[p.ServerAlias]
	if !ok {
		return fmt.Errorf("NATS server alias %s not found", p.ServerAlias)
	}

	msg, err := common.NatsMsgForHttpRequest(r, subj)
	if err != nil {
		return err
	}

	resp, err := server.Conn.RequestMsg(msg, p.Timeout)
	if err != nil {
		return fmt.Errorf("could not request NATS message: %w", err)
	}

	if err == nats.ErrNoResponders {
		w.WriteHeader(http.StatusNotFound)
		p.logger.Warn("No Responders for NATS subject - answering with HTTP Status Not Found.", zap.String("subject", subj))
		return err
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}

	for k, headers := range resp.Header {
		for _, header := range headers {
			w.Header().Add(k, header)
		}
	}
	_, err = w.Write(resp.Data)
	if err != nil {
		return fmt.Errorf("could not write response back to HTTP Writer: %w", err)
	}

	// we are done :)
	return nil
}

var (
	_ caddyhttp.MiddlewareHandler = (*Request)(nil)
	_ caddy.Provisioner           = (*Request)(nil)
	_ caddyfile.Unmarshaler       = (*Request)(nil)
)
