package publish

import (
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

// ParsePublishHandler parses the nats_publish directive. Syntax:
//
//	nats_publish [serverAlias] subject {
//	    [timeout 42ms]
//	}
func ParsePublishHandler(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	var p = Publish{}
	err := p.UnmarshalCaddyfile(h.Dispenser)
	return p, err
}
func (p *Publish) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		if d.CountRemainingArgs() == 2 {
			if !d.Args(&p.ServerAlias, &p.Subject) {
				// should never fail because of the check above for remainingArgs==2
				return d.Errf("Wrong argument count or unexpected line ending after '%s'", d.Val())
			}
		} else {
			if !d.Args(&p.Subject) {
				return d.Errf("Wrong argument count or unexpected line ending after '%s'", d.Val())
			}
		}

		for d.NextBlock(0) {
			switch d.Val() {
			default:
				return d.Errf("unrecognized subdirective: %s", d.Val())
			}
		}
	}

	return nil
}
