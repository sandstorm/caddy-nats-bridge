package body_jetstream

import (
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"net/http"
	"time"
)

type StoreBodyToJetStream struct {
	Bucket string        `json:"bucket,omitempty"`
	TTL    time.Duration `json:"ttl,omitempty"`
}

func (StoreBodyToJetStream) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.store_body_to_jetstream",
		New: func() caddy.Module { return new(StoreBodyToJetStream) },
	}
}

func (sb *StoreBodyToJetStream) ServeHTTP(writer http.ResponseWriter, request *http.Request, handler caddyhttp.Handler) error {
	//TODO implement me
	panic("implement me")
}
