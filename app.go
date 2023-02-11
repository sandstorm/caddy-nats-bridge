package caddynats

import (
	"encoding/json"
	"fmt"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
	"time"
)

func init() {
	caddy.RegisterModule(NatsBridgeApp{})
}

// NatsBridgeApp connects caddy to a NATS server.
//
// NATS is a simple, secure and performant communications system for digital
// systems, services and devices.
type NatsBridgeApp struct {
	Servers     map[string]*NatsServer `json:"servers,omitempty"`
	HandlersRaw []json.RawMessage      `json:"handle,omitempty" caddy:"namespace=nats.handlers inline_key=handler"`

	// Decoded values
	Handlers []Handler `json:"-"`

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

	// if non-empty, large HTTP request bodies not fitting into a NATS message are stored in jetstream KV
	// (for "nats_publish" or "nats_request")
	LargeRequestBodyJetStreamBucketName string `json:"largeRequestBodyJetStreamBucketName,omitempty"`
	largeRequestBodyObjectStore         nats.ObjectStore
	//LargeResponseBodyJetStreamBucketName string `json:"largeResponseBodyJetStreamBucketName,omitempty"`

	conn *nats.Conn
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

	// Set up handlers
	if app.HandlersRaw != nil {
		vals, err := ctx.LoadModule(app, "HandlersRaw")
		if err != nil {
			return fmt.Errorf("loading handler modules: %v", err)
		}

		for _, val := range vals.([]interface{}) {
			app.Handlers = append(app.Handlers, val.(Handler))
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

		server.conn, err = nats.Connect(server.NatsUrl, opts...)
		if err != nil {
			return fmt.Errorf("could not connect to %s : %w", server.NatsUrl, err)
		}

		if server.LargeRequestBodyJetStreamBucketName != "" {
			js, err := server.conn.JetStream()
			if err != nil {
				return fmt.Errorf("could not load JetStream, although largeRequestBodyJetStreamBucketName is configured")
			}
			server.largeRequestBodyObjectStore, err = js.ObjectStore(server.LargeRequestBodyJetStreamBucketName)
			if err == nats.ErrStreamNotFound {
				server.largeRequestBodyObjectStore, err = js.CreateObjectStore(&nats.ObjectStoreConfig{
					Bucket:      server.LargeRequestBodyJetStreamBucketName,
					Description: "Temporary object store for large file uploads",
					TTL:         5 * time.Minute, // TODO: configurable??
				})
				if err != nil {
					return fmt.Errorf("object store %s could not be greated", server.LargeRequestBodyJetStreamBucketName)
				}
			} else if err != nil {
				return fmt.Errorf("could not load ObjectStore %s: %w", server.LargeRequestBodyJetStreamBucketName, err)
			}
		}

		app.logger.Info("connected to NATS server", zap.String("url", server.conn.ConnectedUrlRedacted()))

		// TODO
		/*for _, handler := range app.Handlers {
			err := handler.Subscribe(conn)
			if err != nil {
				return err
			}
		}*/
	}

	return nil
}

func (app *NatsBridgeApp) Stop() error {
	defer func() {
		for _, server := range app.Servers {
			server.conn.Close()
		}
	}()

	for _, server := range app.Servers {
		app.logger.Info("closing NATS connection", zap.String("url", server.conn.ConnectedUrlRedacted()))
		// TODO
		/*for _, handler := range app.Handlers {
			err := handler.Unsubscribe(app.conn)
			if err != nil {
				return err
			}
		}*/
	}

	return nil
}

// Interface guards
var (
	_ caddy.App             = (*NatsBridgeApp)(nil)
	_ caddy.Provisioner     = (*NatsBridgeApp)(nil)
	_ caddyfile.Unmarshaler = (*NatsBridgeApp)(nil)
)
