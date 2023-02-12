package subscribe

import "github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"

func ParseSubscribeHandler(d *caddyfile.Dispenser) (Subscribe, error) {
	s := Subscribe{}
	// TODO: handle errors better here
	if !d.AllArgs(&s.Subject, &s.Method, &s.URL) {
		return s, d.Err("wrong number of arguments")
	}

	return s, nil
}
