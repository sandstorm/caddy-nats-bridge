package caddynats

import (
	"fmt"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
	"net/http"
)

const publishDefaultTimeout = 10000

func init() {
	caddy.RegisterModule(Publish{})
}

type Publish struct {
	Subject string `json:"subject,omitempty"`
	// TODO WithReply   bool   `json:"with_reply,omitempty"`
	Timeout     int64  `json:"timeout,omitempty"`
	ServerAlias string `json:"serverAlias,omitempty"`

	logger *zap.Logger
	app    *NatsBridgeApp
}

func (Publish) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.nats_publish",
		New: func() caddy.Module { return new(Publish) },
	}
}

func (p *Publish) Provision(ctx caddy.Context) error {
	p.logger = ctx.Logger(p)

	natsAppIface, err := ctx.App("nats")
	if err != nil {
		return fmt.Errorf("getting NATS app: %v. Make sure NATS is configured in global options", err)
	}

	p.app = natsAppIface.(*NatsBridgeApp)

	return nil
}

func (p Publish) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	/*repl := r.Context().Value(caddy.ReplacerCtxKey).(*caddy.Replacer)
	addNATSPublishVarsToReplacer(repl, r)

	//TODO: What method is best here? ReplaceAll vs ReplaceWithErr?
	subj := repl.ReplaceAll(p.Subject, "")

	//TODO: Check max msg size

	p.logger.Debug("publishing NATS message", zap.String("subject", subj), zap.Bool("with_reply", p.WithReply), zap.Int64("timeout", p.Timeout))

	if p.WithReply {
		return p.natsRequestReply(subj, r, w)
	}

	// Otherwise. just publish like normal
	msg, err := p.createMsgForRequest(r, subj)
	if err != nil {
		return err
	}

	err = p.app.conn.PublishMsg(msg)
	if err != nil {
		return err
	}
	*/
	return next.ServeHTTP(w, r)
}

func (p *Publish) createMsgForRequest(r *http.Request, subject string) (*nats.Msg, error) {
	/*var msg *nats.Msg
	if r.ContentLength == -1 || r.ContentLength > 950_000_000 {

		// content length unknown => chunked upload
		// Content length > 950 KB

		// => we temporarily store body in temp JetStream Object Store
		fileStreamId := nuid.Next()

		js, err := p.app.conn.JetStream()
		if err != nil {
			return nil, err
		}

		os, err := js.CreateObjectStore(&nats.ObjectStoreConfig{
			Bucket:      "temp-uploadfiles",
			Description: "Temporary",
			TTL:         5 * time.Minute,
		})
		if err != nil {
			return nil, err
		}

		b, _ := io.ReadAll(r.Body) // TODO: directly sending r.Body to NATS does not work for some reason (although it should ^^^
		_, err = os.Put(&nats.ObjectMeta{
			Name: fileStreamId,
		}, bytes.NewReader(b)) // r.Body
		p.logger.Info("large file, putting to JetStream", zap.String("fileStreamId", fileStreamId))
		if err != nil {
			return nil, fmt.Errorf("cannot store binary to Object Store: %w", err)
		}
		msg = &nats.Msg{
			Subject: subject,
			Header:  nats.Header(r.Header),
		}
		msg.Header.Add("X-Large-Body-Id", fileStreamId)
	} else {
		// "small" message -> embedded into Nats msg.
		b, _ := io.ReadAll(r.Body)
		msg = &nats.Msg{
			Subject: subject,
			Header:  nats.Header(r.Header),
			Data:    b,
		}

	}

	msg.Header.Add("X-Http-Method", r.Method)
	msg.Header.Add("X-Http-Url", r.URL.String())
	return msg, nil*/
	return nil, nil
}

//
//func (p *Publish) natsRequestReply(subject string, r *http.Request, w http.ResponseWriter) error {
//	msg, err := p.createMsgForRequest(r, subject)
//	if err != nil {
//		return err
//	}
//	m, err := p.app.conn.RequestMsg(msg, time.Duration(p.Timeout)*time.Millisecond)
//
//	// nats.ErrMaxPayload
//
//	//data, err := io.ReadAll(r.Body)
//	//if err != nil {
//	//	return err
//	//}
//
//	//os.AddLink()
//	// TODO: Make error handlers configurable
//	if err == nats.ErrNoResponders {
//		w.WriteHeader(http.StatusNotFound)
//		return err
//	} else if err != nil {
//		w.WriteHeader(http.StatusInternalServerError)
//		return err
//	}
//
//	_, err = w.Write(m.Data)
//
//	return err
//}

var (
	_ caddyhttp.MiddlewareHandler = (*Publish)(nil)
	_ caddy.Provisioner           = (*Publish)(nil)
	_ caddyfile.Unmarshaler       = (*Publish)(nil)
)
