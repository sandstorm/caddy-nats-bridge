package body_jetstream

import (
	"bytes"
	"fmt"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nuid"
	"go.uber.org/zap"
	"io"
	"net/http"
	"sandstorm.de/custom-caddy/nats-bridge/common"
	"sandstorm.de/custom-caddy/nats-bridge/global"
	"sync/atomic"
	"time"
)

type StoreBodyToJetStream struct {
	Bucket string        `json:"bucket,omitempty"`
	TTL    time.Duration `json:"ttl,omitempty"`
	// in which NATS server should the request body be stored?
	ServerAlias string `json:"serverAlias,omitempty"`

	app    *global.NatsBridgeApp
	logger *zap.Logger
	// always use objectStore() to access, to ensure it is initialized.
	os atomic.Pointer[nats.ObjectStore]
}

func (StoreBodyToJetStream) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.store_body_to_jetstream",
		New: func() caddy.Module { return new(StoreBodyToJetStream) },
	}
}

func (sb *StoreBodyToJetStream) Provision(ctx caddy.Context) error {
	sb.logger = ctx.Logger()

	natsAppIface, err := ctx.App("nats")
	if err != nil {
		return fmt.Errorf("getting NATS app: %w. Make sure NATS is configured in global options", err)
	}

	sb.app = natsAppIface.(*global.NatsBridgeApp)

	return nil
}

func (sb *StoreBodyToJetStream) ServeHTTP(writer http.ResponseWriter, request *http.Request, handler caddyhttp.Handler) error {
	b, err := io.ReadAll(request.Body)
	if err != nil {
		return fmt.Errorf("cannot read request body: %w", err)
	}
	if len(b) > 0 {
		// because HTTP headers are changed in camelization ("X-NatsHttp" will become "X-Natshttp"), we need to store our
		// extra headers in the Request Context. This way, we can ensure the headers are set as they are configured.
		// This wouldn't matter much if it was just internal usage; but we want to expose the header name in config (and
		// it would be very weird if there were additional constraints on the header names)
		extraNatsMsgHeaders := common.ExtraNatsMsgHeadersFromContext(request.Context())
		extraNatsMsgHeaders["X-NatsHttp-Body-Bucket"] = sb.Bucket
		id := nuid.Next()
		extraNatsMsgHeaders["X-NatsHttp-Body-Id"] = id
		request = request.WithContext(extraNatsMsgHeaders.StoreInCtx(request.Context()))

		os, err := sb.objectStore()
		if err != nil {
			return fmt.Errorf("cannot retrieve object store: %w", err)
		}

		_, err = os.Put(&nats.ObjectMeta{
			Name: id,
		}, bytes.NewReader(b)) // TODO: we cannot directly stream request.Body to os.Put, although it should work type-wise - so we read the full resp into bytes
		if err != nil {
			return fmt.Errorf("cannot store binary to Object Store %s: %w", sb.Bucket, err)
		}

		// empty the request body for sub-handlers.
		request.Body = io.NopCloser(bytes.NewReader([]byte{}))
	}

	return handler.ServeHTTP(writer, request)
}

// objectStore is lazily initializing the NATS JetStream object store on first access.
// This is not possible inside Provision(), because we do not know whether the global.NatsBridgeApp
// is already set up or not (because provisioning order is not deterministic).
func (sb *StoreBodyToJetStream) objectStore() (nats.ObjectStore, error) {
	tmp := sb.os.Load()
	if tmp != nil {
		return *tmp, nil
	}

	// set up ObjectStore
	server := sb.app.Servers[sb.ServerAlias]
	js, err := server.Conn.JetStream()
	if err != nil {
		return nil, fmt.Errorf("could not load JetStream: %w", err)
	}
	os, err := js.ObjectStore(sb.Bucket)
	if err == nats.ErrStreamNotFound {
		// Object store does not exist yet, create it.
		sb.logger.Info("Creating object store", zap.String("Bucket", sb.Bucket), zap.Duration("TTL", sb.TTL))
		os, err = js.CreateObjectStore(&nats.ObjectStoreConfig{
			Bucket: sb.Bucket,
			TTL:    sb.TTL,
		})
		if err != nil {
			return nil, fmt.Errorf("could not create ObjectStore for bucket %s: %w", sb.Bucket, err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("could not load ObjectStore for bucket %s: %w", sb.Bucket, err)
	}

	err = sb.checkIfTTLMatchesConfig(os)
	if err != nil {
		return nil, err
	}

	// store the object store reference for all further requests.
	sb.os.Store(&os)

	return os, nil
}

// checkIfTTLMatchesConfig is called during Provision to check if the objectStore's TTL setting matches the configuration.
// we do not auto-update it, because it seemed too complex for now.
func (sb StoreBodyToJetStream) checkIfTTLMatchesConfig(os nats.ObjectStore) error {
	st, err := os.Status()
	if err != nil {
		return fmt.Errorf("could not read ObjectStore Status for bucket %s: %w", sb.Bucket, err)
	}

	if st.TTL() != sb.TTL {
		return fmt.Errorf("object store %s TTL Mismatch. Current TTL: %d. Configured TTL: %d", sb.Bucket, st.TTL(), sb.TTL)
	}

	return nil
}

var (
	_ caddyhttp.MiddlewareHandler = (*StoreBodyToJetStream)(nil)
	_ caddy.Provisioner           = (*StoreBodyToJetStream)(nil)
	//_ caddyfile.Unmarshaler       = (*StoreBodyToJetStream)(nil)
)
