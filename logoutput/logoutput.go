package logoutput

import (
	"fmt"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/nats-io/nats.go"
	"github.com/sandstorm/caddy-nats-bridge/natsbridge"
	"go.uber.org/zap"
	"io"
)

type LogOutput struct {
	Subject     string `json:"subject,omitempty"`
	ServerAlias string `json:"serverAlias,omitempty"`

	logger   *zap.Logger
	caddyCtx caddy.Context
}

func (LogOutput) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "caddy.logging.writers.nats",
		New: func() caddy.Module {
			// Default values
			return &LogOutput{
				ServerAlias: "default",
			}
		},
	}
}

func (p *LogOutput) Provision(ctx caddy.Context) error {
	p.logger = ctx.Logger(p)
	p.caddyCtx = ctx

	return nil
}

func (p LogOutput) String() string {
	return fmt.Sprintf("nats(server=%s, subject=%s)", p.ServerAlias, p.Subject)
}

func (p LogOutput) WriterKey() string {
	return fmt.Sprintf("nats-%s%s", p.ServerAlias, p.Subject)
}

func (p LogOutput) OpenWriter() (io.WriteCloser, error) {
	return LogOutputWriter{
		logOutput: p,
	}, nil
}

type LogOutputWriter struct {
	logOutput LogOutput
	natsConn  *nats.Conn
}

func (lw LogOutputWriter) Write(msg []byte) (n int, err error) {
	// NOTE: we are only allowed to lazily initialize the natsConn from server.Conn, because caddyCtx.App() crashes
	// when called inside Provision(). This is because:
	// in caddy.go, function "run", first, logging is initialized via "newCfg.Logging.openLogs(ctx)", and then
	// the "apps" key is initialized via "newCfg.apps = make(map[string]App)".
	// calling caddyCtx.App("nats") will crash in case newCfg.apps is not properly initialized as Map yet.
	//
	// => WORKAROUND: we fetch the natsConnection here, when sending the 1st log message.
	if lw.natsConn == nil {
		natsAppIface, err := lw.logOutput.caddyCtx.App("nats")
		if err != nil {
			return 0, fmt.Errorf("getting NATS app: %w. Make sure NATS is configured in nats options", err)
		}
		app := natsAppIface.(*natsbridge.NatsBridgeApp)
		server, ok := app.Servers[lw.logOutput.ServerAlias]
		if !ok {
			return 0, fmt.Errorf("NATS server alias %s not found", lw.logOutput.ServerAlias)
		}
		lw.natsConn = server.Conn
	}

	err = lw.natsConn.Publish(lw.logOutput.Subject, msg)
	if err != nil {
		return 0, fmt.Errorf("error writing log message: %w", err)
	}
	return len(msg), nil
}

func (lw LogOutputWriter) Close() error {
	// nothing to be done
	return nil
}

var (
	_ caddy.WriterOpener    = (*LogOutput)(nil)
	_ caddy.Provisioner     = (*LogOutput)(nil)
	_ caddyfile.Unmarshaler = (*LogOutput)(nil)
	_ io.WriteCloser        = (*LogOutputWriter)(nil)
)
