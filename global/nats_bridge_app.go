package global

import (
	"encoding/json"
	"fmt"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
	"sandstorm.de/custom-caddy/nats-bridge/common"
)

// NatsBridgeApp is the global nats bridge for Caddy.
//
// NATS is a simple, secure and performant communications system for digital
// systems, services and devices.
type NatsBridgeApp struct {
	// Immutable after provisioning
	Servers map[string]*NatsServer `json:"servers,omitempty"`

	logger *zap.Logger
	ctx    caddy.Context
}

type NatsServer struct {
	// can also contain comma-separated list of URLs, see nats.Connect
	NatsUrl            string `json:"url,omitempty"`
	UserCredentialFile string `json:"userCredentialFile,omitempty"`
	NkeyCredentialFile string `json:"nkeyCredentialFile,omitempty"`
	ClientName         string `json:"clientName,omitempty"`
	InboxPrefix        string `json:"inboxPrefix,omitempty"`

	HandlersRaw []json.RawMessage `json:"handle,omitempty" caddy:"namespace=nats.handlers inline_key=handler"`

	// Decoded values
	Handlers []common.NatsHandler `json:"-"`

	Conn *nats.Conn `json:"-"`
}

// CaddyModule returns the Caddy module information.
func (app NatsBridgeApp) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "nats",
		New: func() caddy.Module {
			app := new(NatsBridgeApp)
			app.Servers = make(map[string]*NatsServer)
			return app
		},
	}
}

// Provision sets up the app
func (app *NatsBridgeApp) Provision(ctx caddy.Context) error {
	// Set logger and NatsUrl
	app.ctx = ctx
	app.logger = ctx.Logger(app)

	// Set up handlers for each server
	for _, server := range app.Servers {
		if server.HandlersRaw != nil {
			vals, err := ctx.LoadModule(server, "HandlersRaw")
			if err != nil {
				return fmt.Errorf("loading handler modules: %v", err)
			}
			for _, val := range vals.([]interface{}) {
				server.Handlers = append(server.Handlers, val.(common.NatsHandler))
			}
		}
	}

	return nil
}

func (app *NatsBridgeApp) Start() error {
	for _, server := range app.Servers {
		// Connect to the NATS server
		app.logger.Info("connecting via NATS URL: ", zap.String("natsUrl", server.NatsUrl))

		var err error
		var opts []nats.Option

		if server.ClientName != "" {
			opts = append(opts, nats.Name(server.ClientName))
		}
		if server.InboxPrefix != "" {
			opts = append(opts, nats.CustomInboxPrefix(server.InboxPrefix))
		}

		if server.UserCredentialFile != "" {
			// JWT
			opts = append(opts, nats.UserCredentials(server.UserCredentialFile))
		} else if server.NkeyCredentialFile != "" {
			// NKEY
			opt, err := nats.NkeyOptionFromSeed(server.NkeyCredentialFile)
			if err != nil {
				return fmt.Errorf("could not load NKey from %s: %w", server.NkeyCredentialFile, err)
			}
			opts = append(opts, opt)
		}

		server.Conn, err = nats.Connect(server.NatsUrl, opts...)
		if err != nil {
			return fmt.Errorf("could not connect to %s : %w", server.NatsUrl, err)
		}

		app.logger.Info("connected to NATS server", zap.String("url", server.Conn.ConnectedUrlRedacted()))

		for _, handler := range server.Handlers {
			err := handler.Subscribe(server.Conn)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (app *NatsBridgeApp) Stop() error {
	defer func() {
		for _, server := range app.Servers {
			app.logger.Info("closing NATS connection", zap.String("url", server.Conn.ConnectedUrlRedacted()))
			server.Conn.Close()
		}
	}()

	app.logger.Info("stopping all NATS subscriptions")
	for _, server := range app.Servers {
		for _, handler := range server.Handlers {
			err := handler.Unsubscribe(server.Conn)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Interface guards
var (
	_ caddy.App             = (*NatsBridgeApp)(nil)
	_ caddy.Provisioner     = (*NatsBridgeApp)(nil)
	_ caddyfile.Unmarshaler = (*NatsBridgeApp)(nil)
)
