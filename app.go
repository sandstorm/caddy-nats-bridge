package caddynats

import (
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"sandstorm.de/custom-caddy/nats-bridge/body_jetstream"
	"sandstorm.de/custom-caddy/nats-bridge/global"
	"sandstorm.de/custom-caddy/nats-bridge/publish"
	"sandstorm.de/custom-caddy/nats-bridge/subscribe"
)

func init() {
	caddy.RegisterModule(global.NatsBridgeApp{})
	httpcaddyfile.RegisterGlobalOption("nats", global.ParseGobalNatsOption)
	caddy.RegisterModule(subscribe.Subscribe{})
	httpcaddyfile.RegisterHandlerDirective("nats_publish", publish.ParsePublishHandler)

	caddy.RegisterModule(publish.Publish{})
	//httpcaddyfile.RegisterHandlerDirective("nats_publish", publish.ParsePublishHandler)

	// store request body to Jetstream
	caddy.RegisterModule(body_jetstream.StoreBodyToJetStream{})
	httpcaddyfile.RegisterHandlerDirective("store_body_to_jetstream", body_jetstream.ParseStoreBodyToJetstream)
}

// NatsBridgeApp connects caddy to a NATS server.
//
