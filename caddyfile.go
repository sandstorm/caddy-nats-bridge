package caddynats

import (
	"encoding/json"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

func init() {
	httpcaddyfile.RegisterGlobalOption("nats", parseGobalNatsOption)
	httpcaddyfile.RegisterHandlerDirective("nats_publish", parsePublishHandler)
	//httpcaddyfile.RegisterHandlerDirective("nats_request", parseRequestHandler)
}

func parseGobalNatsOption(d *caddyfile.Dispenser, existingVal interface{}) (interface{}, error) {
	app := new(NatsBridgeApp)
	app.Servers = make(map[string]*NatsServer)

	if existingVal != nil {
		var ok bool
		caddyFileApp, ok := existingVal.(httpcaddyfile.App)
		if !ok {
			return nil, d.Errf("existing nats values of unexpected type: %T", existingVal)
		}
		err := json.Unmarshal(caddyFileApp.Value, app)
		if err != nil {
			return nil, err
		}
	}

	err := app.UnmarshalCaddyfile(d)

	return httpcaddyfile.App{
		Name:  "nats",
		Value: caddyconfig.JSON(app, nil),
	}, err
}

/*func parseRequestHandler(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	var p = Publish{
		WithReply: true,
		Timeout:   publishDefaultTimeout,
	}
	err := p.UnmarshalCaddyfile(h.Dispenser)
	return p, err
}

func parseSubscribeHandler(d *caddyfile.Dispenser) (Subscribe, error) {
	s := Subscribe{}
	// TODO: handle errors better here
	if !d.AllArgs(&s.Subject, &s.Method, &s.URL) {
		return s, d.Err("wrong number of arguments")
	}

	return s, nil
}*/

/*func parseQueueSubscribeHandler(d *caddyfile.Dispenser) (Subscribe, error) {
	s := Subscribe{}
	// TODO: handle errors better here
	if !d.AllArgs(&s.Subject, &s.QueueGroup, &s.Method, &s.URL) {
		return s, d.Err("wrong number of arguments")
	}

	return s, nil
}*/

func (app *NatsBridgeApp) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		// parse the server alias and fall back to "default"
		serverAlias := "default"
		if d.NextArg() {
			serverAlias = d.Val()
		}
		server, ok := app.Servers[serverAlias]
		if ok == false {
			server = &NatsServer{}
			app.Servers[serverAlias] = server
		}
		if d.NextArg() {
			return d.ArgErr()
		}

		for nesting := d.Nesting(); d.NextBlock(nesting); {
			switch d.Val() {
			case "url":
				if !d.AllArgs(&server.NatsUrl) {
					return d.ArgErr()
				}
			case "userCredentialFile":
				if !d.AllArgs(&server.UserCredentialFile) {
					return d.ArgErr()
				}
			case "nkeyCredentialFile":
				if !d.AllArgs(&server.NkeyCredentialFile) {
					return d.ArgErr()
				}
			case "clientName":
				if !d.AllArgs(&server.ClientName) {
					return d.ArgErr()
				}
			case "inboxPrefix":
				if !d.AllArgs(&server.InboxPrefix) {
					return d.ArgErr()
				}

			/*case "subscribe":
				s, err := parseSubscribeHandler(d)
				if err != nil {
					return err
				}
				jsonHandler := caddyconfig.JSONModuleObject(s, "handler", s.CaddyModule().ID.Name(), nil)
				app.HandlersRaw = append(app.HandlersRaw, jsonHandler)

			case "reply":
				s, err := parseSubscribeHandler(d)
				s.WithReply = true
				if err != nil {
					return err
				}
				jsonHandler := caddyconfig.JSONModuleObject(s, "handler", s.CaddyModule().ID.Name(), nil)
				app.HandlersRaw = append(app.HandlersRaw, jsonHandler)

			case "queue_subscribe":
				s, err := parseQueueSubscribeHandler(d)
				if err != nil {
					return err
				}
				jsonHandler := caddyconfig.JSONModuleObject(s, "handler", s.CaddyModule().ID.Name(), nil)
				app.HandlersRaw = append(app.HandlersRaw, jsonHandler)

			case "queue_reply":
				s, err := parseQueueSubscribeHandler(d)
				s.WithReply = true
				if err != nil {
					return err
				}
				jsonHandler := caddyconfig.JSONModuleObject(s, "handler", s.CaddyModule().ID.Name(), nil)
				app.HandlersRaw = append(app.HandlersRaw, jsonHandler)*/

			default:
				return d.Errf("unrecognized subdirective: %s", d.Val())
			}
		}
	}

	return nil
}

func parsePublishHandler(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	var p = Publish{
		//WithReply: false,
		//Timeout:     publishDefaultTimeout,
		ServerAlias: "default",
	}
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
			/*case "timeout":
			if !d.NextArg() {
				return d.ArgErr()
			}
			t, err := strconv.Atoi(d.Val())
			if err != nil {
				return d.Err("timeout is not a valid integer")
			}

			p.Timeout = int64(t)*/
			default:
				return d.Errf("unrecognized subdirective: %s", d.Val())
			}
		}
	}

	return nil
}
