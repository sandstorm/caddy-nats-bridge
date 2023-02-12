package subscribe

import (
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
)

// ParseSubscribeHandler parses the subscribe directive. Syntax:
//
//	subscribe subjectPattern HTTPMethod HTTPURL {
//	    [queue queueGroupName]
//	}
func ParseSubscribeHandler(d *caddyfile.Dispenser) (*Subscribe, error) {
	s := Subscribe{}
	if !d.Args(&s.Subject, &s.Method, &s.URL) {
		return nil, d.ArgErr()
	}

	for nesting := d.Nesting(); d.NextBlock(nesting); {
		switch d.Val() {
		case "queue":
			if !d.AllArgs(&s.QueueGroup) {
				return nil, d.ArgErr()
			}
		default:
			return nil, d.Errf("unrecognized subdirective: %s", d.Val())
		}
	}

	return &s, nil
}
