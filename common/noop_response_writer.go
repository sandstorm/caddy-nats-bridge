package common

import "net/http"

type NoopResponseWriter struct {
	headers http.Header
}

func (NoopResponseWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

func (NoopResponseWriter) WriteHeader(statusCode int) {
	//noop
}

func (n NoopResponseWriter) Header() http.Header {
	if n.headers == nil {
		n.headers = http.Header{}
	}
	return n.headers
}
