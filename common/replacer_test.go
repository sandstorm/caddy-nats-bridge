package common

import (
	"bytes"
	"net/http"
	"reflect"
	"testing"

	"github.com/caddyserver/caddy/v2"
	"github.com/nats-io/nats.go"
)

// Returns a basic request for testing
func reqPath(path string) *http.Request {
	req, _ := http.NewRequest("GET", "http://localhost"+path, &bytes.Buffer{})
	return req
}

func TestAddNatsPublishVarsToReplacer(t *testing.T) {
	type test struct {
		req *http.Request

		input string
		want  string
	}

	tests := []test{

		// Basic subject mapping
		{req: reqPath("/foo/bar"), input: "{http.request.uri.path.asNatsSubject}", want: "foo.bar"},
		{req: reqPath("/foo/bar/bat/baz"), input: "{http.request.uri.path.asNatsSubject}", want: "foo.bar.bat.baz"},
		{req: reqPath("/foo/bar/bat/baz?query=true"), input: "{http.request.uri.path.asNatsSubject}", want: "foo.bar.bat.baz"},
		{req: reqPath("/foo/bar"), input: "prefix.{http.request.uri.path.asNatsSubject}.suffix", want: "prefix.foo.bar.suffix"},

		// Segment placeholders
		{req: reqPath("/foo/bar"), input: "{http.request.uri.path.asNatsSubject.0}", want: "foo"},
		{req: reqPath("/foo/bar"), input: "{http.request.uri.path.asNatsSubject.1}", want: "bar"},

		// Segment Ranges
		{req: reqPath("/foo/bar/bat/baz"), input: "{http.request.uri.path.asNatsSubject.0:}", want: "foo.bar.bat.baz"},
		{req: reqPath("/foo/bar/bat/baz"), input: "{http.request.uri.path.asNatsSubject.1:}", want: "bar.bat.baz"},
		{req: reqPath("/foo/bar/bat/baz"), input: "{http.request.uri.path.asNatsSubject.2:}", want: "bat.baz"},
		{req: reqPath("/foo/bar/bat/baz"), input: "{http.request.uri.path.asNatsSubject.1:3}", want: "bar.bat"},
		{req: reqPath("/foo/bar/bat/baz"), input: "{http.request.uri.path.asNatsSubject.0:3}", want: "foo.bar.bat"},
		{req: reqPath("/foo/bar/bat/baz"), input: "{http.request.uri.path.asNatsSubject.0:4}", want: "foo.bar.bat.baz"},
		{req: reqPath("/foo/bar/bat/baz"), input: "{http.request.uri.path.asNatsSubject.:3}", want: "foo.bar.bat"},

		// Out of bounds ranges
		{req: reqPath("/foo/bar/bat/baz"), input: "{http.request.uri.path.asNatsSubject.0:18}", want: ""},
		{req: reqPath("/foo/bar/bat/baz"), input: "{http.request.uri.path.asNatsSubject.-1:}", want: ""},
	}

	for _, tc := range tests {
		repl := caddy.NewReplacer()
		AddNATSPublishVarsToReplacer(repl, tc.req)
		got := repl.ReplaceAll(tc.input, "")
		if !reflect.DeepEqual(tc.want, got) {
			t.Errorf("expected: %v, got: %v", tc.want, got)
		}
	}
}

func TestAddNatsSubscribeVarsToReplacer(t *testing.T) {
	type test struct {
		msg *nats.Msg

		input string
		want  string
	}

	tests := []test{
		// Basic subject mapping
		{msg: nats.NewMsg("foo.bar"), input: "{nats.request.subject.asUriPath}", want: "foo/bar"},
		{msg: nats.NewMsg("foo.bar.bat.baz"), input: "{nats.request.subject.asUriPath}", want: "foo/bar/bat/baz"},
		{msg: nats.NewMsg("foo.bar"), input: "prefix/{nats.request.subject.asUriPath}/suffix", want: "prefix/foo/bar/suffix"},

		// // Segment placeholders
		{msg: nats.NewMsg("foo.bar"), input: "{nats.request.subject.0}", want: "foo"},
		{msg: nats.NewMsg("foo.bar"), input: "{nats.request.subject.1}", want: "bar"},
		{msg: nats.NewMsg("foo.bar"), input: "{nats.request.subject.asUriPath.0}", want: "foo"},
		{msg: nats.NewMsg("foo.bar"), input: "{nats.request.subject.asUriPath.1}", want: "bar"},
		// TODO: nats.request.subject.0 etc -> also allow this.

		// // Segment Ranges
		{msg: nats.NewMsg("foo.bar.bat.baz"), input: "{nats.request.subject.0:}", want: "foo.bar.bat.baz"},
		{msg: nats.NewMsg("foo.bar.bat.baz"), input: "{nats.request.subject.1:}", want: "bar.bat.baz"},
		{msg: nats.NewMsg("foo.bar.bat.baz"), input: "{nats.request.subject.2:}", want: "bat.baz"},
		{msg: nats.NewMsg("foo.bar.bat.baz"), input: "{nats.request.subject.1:3}", want: "bar.bat"},
		{msg: nats.NewMsg("foo.bar.bat.baz"), input: "{nats.request.subject.0:3}", want: "foo.bar.bat"},
		{msg: nats.NewMsg("foo.bar.bat.baz"), input: "{nats.request.subject.:3}", want: "foo.bar.bat"},
		{msg: nats.NewMsg("foo.bar.bat.baz"), input: "{nats.request.subject.asUriPath.0:}", want: "foo/bar/bat/baz"},
		{msg: nats.NewMsg("foo.bar.bat.baz"), input: "{nats.request.subject.asUriPath.1:}", want: "bar/bat/baz"},
		{msg: nats.NewMsg("foo.bar.bat.baz"), input: "{nats.request.subject.asUriPath.2:}", want: "bat/baz"},
		{msg: nats.NewMsg("foo.bar.bat.baz"), input: "{nats.request.subject.asUriPath.1:3}", want: "bar/bat"},
		{msg: nats.NewMsg("foo.bar.bat.baz"), input: "{nats.request.subject.asUriPath.0:3}", want: "foo/bar/bat"},
		{msg: nats.NewMsg("foo.bar.bat.baz"), input: "{nats.request.subject.asUriPath.:3}", want: "foo/bar/bat"},

		// Out of bounds ranges
		{msg: nats.NewMsg("foo.bar.bat.baz"), input: "{nats.request.subject.0:18}", want: ""},
		{msg: nats.NewMsg("foo.bar.bat.baz"), input: "{nats.request.subject.:18}", want: ""},
		{msg: nats.NewMsg("foo.bar.bat.baz"), input: "{nats.request.subject.-1:}", want: ""},
		{msg: nats.NewMsg("foo.bar.bat.baz"), input: "{nats.request.subject.asUriPath.0:18}", want: ""},
		{msg: nats.NewMsg("foo.bar.bat.baz"), input: "{nats.request.subject.asUriPath.:18}", want: ""},
		{msg: nats.NewMsg("foo.bar.bat.baz"), input: "{nats.request.subject.asUriPath.-1:}", want: ""},
	}

	for _, tc := range tests {
		repl := caddy.NewReplacer()
		AddNatsSubscribeVarsToReplacer(repl, tc.msg)
		got := repl.ReplaceAll(tc.input, "")
		if !reflect.DeepEqual(tc.want, got) {
			t.Errorf("expected: %v, got: %v. Input: %s", tc.want, got, tc.input)
		}
	}
}
