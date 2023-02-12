package global

import (
	"encoding/json"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
)

func ParseGobalNatsOption(d *caddyfile.Dispenser, existingVal interface{}) (interface{}, error) {
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
