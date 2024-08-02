package common

import (
	"io"
	"net/http"

	"github.com/nats-io/nats.go"
)

// NatsMsgForHttpRequest creates a nats.Msg from an existing http.Request: the HTTP Request Body is transferred
// to the NATS message Data, and the headers are transferred as well if 'headers' is true.
//
// Three special headers are added for the request method, URL path, and raw query.
func NatsMsgForHttpRequest(r *http.Request, subject string, headers bool) (*nats.Msg, error) {
	var msg *nats.Msg
	b, _ := io.ReadAll(r.Body)

	if headers {
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

	} else {
		msg = &nats.Msg{
			Subject: subject,
			Data:    b,
		}
	}

	return msg, nil
}
