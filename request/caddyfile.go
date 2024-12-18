package request

import (
	"strconv"
	"time"

	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

// ParseRequestHandler parses the nats_request directive. Syntax:
//
//	nats_request [serverAlias] subject {
//	    [timeout 1s]
//      [headers true|false]
//	}
func ParseRequestHandler(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	var p = Request{}
	err := p.UnmarshalCaddyfile(h.Dispenser)
	return p, err
}
func (p *Request) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
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
			case "timeout":
				if !d.NextArg() {
					return d.ArgErr()
				}
				t, err := time.ParseDuration(d.Val())
				if err != nil {
					return d.Err("timeout is not a valid duration")
				}

				p.Timeout = t

			case "headers":
				if !d.NextArg() {
					return d.ArgErr()
				}
				h, err := strconv.ParseBool(d.Val())
				if err != nil {
					return d.Err("headers is not a boolean")
				}

				p.Headers = h

			default:
				return d.Errf("unrecognized subdirective: %s", d.Val())
			}
		}
	}

	return nil
}
