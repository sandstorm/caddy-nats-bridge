package logoutput

import (
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
)

func (p *LogOutput) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
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
			/*case "timeout":
			if !d.NextArg() {
				return d.ArgErr()
			}
			t, err := time.ParseDuration(d.Val())
			if err != nil {
				return d.Err("timeout is not a valid duration")
			}

			p.Timeout = t*/
			default:
				return d.Errf("unrecognized subdirective: %s", d.Val())
			}
		}
	}

	return nil
}
