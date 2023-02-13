package common

import (
	"github.com/nats-io/nats.go"
	"io"
	"net/http"
)

func NatsMsgForHttpRequest(r *http.Request, subject string) (*nats.Msg, error) {
	var msg *nats.Msg
	b, _ := io.ReadAll(r.Body)

	headers := nats.Header(r.Header)
	for k, v := range ExtraNatsMsgHeadersFromContext(r.Context()) {
		headers.Add(k, v)
	}
	msg = &nats.Msg{
		Subject: subject,
		Header:  headers,
		Data:    b,
	}

	msg.Header.Add("X-NatsBridge-Method", r.Method)
	msg.Header.Add("X-NatsBridge-UrlPath", r.URL.Path)
	msg.Header.Add("X-NatsBridge-UrlQuery", r.URL.RawQuery)
	return msg, nil
}
