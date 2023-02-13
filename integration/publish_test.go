package integration

import (
	"bufio"
	"fmt"
	"github.com/caddyserver/caddy/v2/caddytest"
	"github.com/nats-io/nats.go"
	_ "github.com/sandstorm/caddy-nats-bridge"
	"net/http"
	"strings"

	"testing"
	"time"
)

func failOnErr(message string, err error, t *testing.T) {
	t.Helper()
	if err != nil {
		t.Fatalf(message, err)
	}
}

const defaultCaddyConf = `
{
	http_port 8889
	admin localhost:2999
	nats {
		url 127.0.0.1:8369
		%s
	}
}
`

// TestPublishToNats converts a HTTP request to a NATS Publication.
// It does not expect a response.
//
//	              ┌──────────────┐    HTTP: /test
//	◀─────────────│ Caddy /test  │◀───────
//	NATS subject  │ nats_publish │
//	 greet.*      └──────────────┘
func TestPublishToNats(t *testing.T) {
	type testCase struct {
		description       string
		buildHttpRequest  func(t *testing.T) *http.Request
		assertNatsMessage func(msg *nats.Msg, nc *nats.Conn, t *testing.T)
		CaddyfileSnippet  string
	}

	// Testcases
	cases := []testCase{
		{
			description: "Simple GET request should keep headers and contain extra X-NatsBridge-Method and X-NatsBridge-UrlPath",
			buildHttpRequest: func(t *testing.T) *http.Request {
				req, err := http.NewRequest("GET", "http://localhost:8889/test/hi", nil)
				failOnErr("Error creating request: %w", err, t)

				req.Header.Add("Custom-Header", "MyValue")
				return req
			},
			CaddyfileSnippet: `
				route /test/* {
					nats_publish greet.hello
				}
			`,
			assertNatsMessage: func(msg *nats.Msg, nc *nats.Conn, t *testing.T) {
				if msg.Header.Get("Custom-Header") != "MyValue" {
					t.Fatalf("Custom-Header not correct, expected 'MyValue', actual headers: %+v", msg.Header)
				}

				if msg.Header.Get("X-NatsBridge-Method") != "GET" {
					t.Fatalf("X-NatsBridge-Method not correct, expected 'GET', actual headers: %+v", msg.Header)
				}
				if msg.Header.Get("X-NatsBridge-UrlPath") != "/test/hi" {
					t.Fatalf("X-NatsBridge-UrlPath not correct, expected '/test/hi', actual headers: %+v", msg.Header)
				}
				if msg.Header.Get("X-NatsBridge-UrlQuery") != "" {
					t.Fatalf("X-NatsBridge-UrlQuery not correct, expected '', actual headers: %+v", msg.Header)
				}
			},
		},
		{
			description: "Request with query parameters should contain extra header",
			buildHttpRequest: func(t *testing.T) *http.Request {
				req, err := http.NewRequest("GET", "http://localhost:8889/test/hi?foo=bar&baz=test", nil)
				failOnErr("Error creating request: %w", err, t)
				return req
			},
			CaddyfileSnippet: `
				route /test/* {
					nats_publish greet.hello
				}
			`,
			assertNatsMessage: func(msg *nats.Msg, nc *nats.Conn, t *testing.T) {
				if msg.Header.Get("X-NatsBridge-UrlQuery") != "foo=bar&baz=test" {
					t.Fatalf("X-NatsBridge-UrlQuery not correct, expected 'foo=bar&baz=test', actual headers: %+v", msg.Header)
				}
			},
		},
		{
			description: "small POST request should contain body",
			buildHttpRequest: func(t *testing.T) *http.Request {
				req, err := http.NewRequest("POST", "http://localhost:8889/test/hi", strings.NewReader("Small Request Body"))
				failOnErr("Error creating request: %w", err, t)
				return req
			},
			CaddyfileSnippet: `
				route /test/* {
					nats_publish greet.hello
				}
			`,
			assertNatsMessage: func(msg *nats.Msg, nc *nats.Conn, t *testing.T) {

				if msg.Header.Get("X-NatsBridge-Method") != "POST" {
					t.Fatalf("X-NatsBridge-Method not correct, expected 'POST', actual headers: %+v", msg.Header)
				}
				if msg.Header.Get("X-NatsBridge-UrlPath") != "/test/hi" {
					t.Fatalf("X-NatsBridge-UrlPath not correct, expected '/test/hi', actual headers: %+v", msg.Header)
				}
				if string(msg.Data) != "Small Request Body" {
					t.Fatalf("Request Body not part of Data. Actual data: %+v", string(msg.Data))
				}
			},
		},
		{
			description: "small POST request should contain body in message payload if it is submitted with Transfer-Encoding: chunked",
			buildHttpRequest: func(t *testing.T) *http.Request {
				// NOTE: we need to use bufio.NewReader, to enforce a Transfer-Encoding=chunked. See net.http.NewRequestWithContext from Go Stdlib.
				req, err := http.NewRequest("POST", "http://localhost:8889/test/hi", bufio.NewReader(strings.NewReader("Small Request Body, but chunked transfer encoding")))
				req.Header.Add("Transfer-Encoding", "chunked")
				failOnErr("Error creating request: %w", err, t)
				return req
			},
			CaddyfileSnippet: `
				route /test/* {
					nats_publish greet.hello
				}
			`,
			assertNatsMessage: func(msg *nats.Msg, nc *nats.Conn, t *testing.T) {
				if msg.Header.Get("X-NatsBridge-Method") != "POST" {
					t.Fatalf("X-NatsBridge-Method not correct, expected 'POST', actual headers: %+v", msg.Header)
				}
				if msg.Header.Get("X-NatsBridge-UrlPath") != "/test/hi" {
					t.Fatalf("X-NatsBridge-UrlPath not correct, expected '/test/hi', actual headers: %+v", msg.Header)
				}
				expected := "Small Request Body, but chunked transfer encoding"
				if string(msg.Data) != expected {
					t.Fatalf("Request Body not part of Data. Actual data: %+v", string(msg.Data))
				}
				if len(msg.Header.Get("X-NatsBridge-LargeBody-Id")) != 0 {
					t.Fatalf("X-NatsBridge-LargeBody-Id should not be set, but was set.")
				}
			},
		},
		// WILDCARDS!!
	}

	// we share the same NATS Server and Caddy Server for all testcases
	_, nc := StartTestNats(t)
	caddyTester := caddytest.NewTester(t)

	for _, testcase := range cases {
		t.Run(testcase.description, func(t *testing.T) {

			subscription, err := nc.SubscribeSync("greet.>")
			defer subscription.Unsubscribe()
			failOnErr("error subscribing to greet.>: %w", err, t)

			caddyTester.InitServer(fmt.Sprintf(defaultCaddyConf+`
				:8889 {
					%s
				}
			`, "", testcase.CaddyfileSnippet), "caddyfile")

			// Build request, and
			req := testcase.buildHttpRequest(t)
			caddyTester.AssertResponse(req, 200, "")
			msg, err := subscription.NextMsg(10 * time.Millisecond)

			if err != nil {
				t.Fatalf("message not received: %v", err)
			} else {
				t.Logf("Received message: %+v", msg)
			}
			testcase.assertNatsMessage(msg, nc, t)
		})
	}
}
