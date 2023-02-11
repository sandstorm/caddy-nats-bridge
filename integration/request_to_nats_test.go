package integration

import (
	"bufio"
	"github.com/caddyserver/caddy/v2/caddytest"
	"github.com/nats-io/nats.go"
	"net/http"
	_ "sandstorm.de/custom-caddy/nats-bridge"
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

func objectStore(nc *nats.Conn, t *testing.T) nats.ObjectStore {
	js, err := nc.JetStream()
	failOnErr("JetStream not retrievable: %w", err, t)

	os, err := js.ObjectStore("temp-uploadfiles")
	failOnErr("Object Store not retrievable: %w", err, t)

	return os
}

const defaultCaddyConf = `
{
	http_port 8889
	admin localhost:2999
	nats 127.0.0.1:8369
}
`

// TestPublishRequestToNats converts a HTTP request to a NATS Publication.
// It does not expect a response.
//
//	        ┌──────────────┐    ┌──────────────┐
//	───────▶│ Caddy /test  │───▶│ NATS subject │──────▶
//	        │ nats_publish │    │   greet.*    │
//	        └──────────────┘    └──────────────┘
func TestPublishRequestToNats(t *testing.T) {
	type testCase struct {
		description       string
		buildHttpRequest  func(t *testing.T) *http.Request
		assertNatsMessage func(msg *nats.Msg, nc *nats.Conn, t *testing.T)
	}

	// Testcases
	cases := []testCase{
		{
			description: "Simple GET request should keep headers and contain extra X-URL",
			buildHttpRequest: func(t *testing.T) *http.Request {
				req, err := http.NewRequest("GET", "http://localhost:8889/test/hi", nil)
				failOnErr("Error creating request: %w", err, t)

				req.Header.Add("Custom-Header", "MyValue")
				return req
			},
			assertNatsMessage: func(msg *nats.Msg, nc *nats.Conn, t *testing.T) {
				if msg.Header.Get("Custom-Header") != "MyValue" {
					t.Fatalf("Custom-Header not correct, expected 'MyValue', actual headers: %+v", msg.Header)
				}

				// TODO: X-URL
				if msg.Header.Get("X-Http-Method") != "GET" {
					t.Fatalf("X-Http-Method not correct, expected 'GET', actual headers: %+v", msg.Header)
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
			assertNatsMessage: func(msg *nats.Msg, nc *nats.Conn, t *testing.T) {
				// TODO: X-URL
				if msg.Header.Get("X-Http-Method") != "POST" {
					t.Fatalf("X-Http-Method not correct, expected 'POST', actual headers: %+v", msg.Header)
				}
				if string(msg.Data) != "Small Request Body" {
					t.Fatalf("Request Body not part of Data. Actual data: %+v", string(msg.Data))
				}
			},
		},
		{
			description: "small POST request should contain body in JetStream and X-Large-Body-Id if it is submitted with Transfer-Encoding: chunked",
			buildHttpRequest: func(t *testing.T) *http.Request {
				// NOTE: we need to use bufio.NewReader, to enforce a Transfer-Encoding=chunked. See net.http.NewRequestWithContext from Go Stdlib.
				req, err := http.NewRequest("POST", "http://localhost:8889/test/hi", bufio.NewReader(strings.NewReader("Small Request Body, but chunked transfer encoding")))
				req.Header.Add("Transfer-Encoding", "chunked")
				failOnErr("Error creating request: %w", err, t)
				return req
			},
			assertNatsMessage: func(msg *nats.Msg, nc *nats.Conn, t *testing.T) {
				// TODO: X-URL
				if msg.Header.Get("X-Http-Method") != "POST" {
					t.Fatalf("X-Http-Method not correct, expected 'POST', actual headers: %+v", msg.Header)
				}
				if len(msg.Data) > 0 {
					t.Fatalf("Request Body should be empty. Actual data: %+v", string(msg.Data))
				}
				if len(msg.Header.Get("X-Large-Body-Id")) == 0 {
					t.Fatalf("X-Large-Body-Id not set or empty.")
				}
				// Read from X-Large-Body-Id
				js, err := nc.JetStream()
				failOnErr("Error getting JetStream Client: %s", err, t)
				os, err := js.ObjectStore("temp-uploadfiles")
				failOnErr("Error getting ObjectStore: %s", err, t)
				resBytes, err := os.GetBytes(msg.Header.Get("X-Large-Body-Id"))
				failOnErr("Error getting Key from ObjectStore: %s", err, t)

				expected := "Small Request Body, but chunked transfer encoding"
				if string(resBytes) != expected {
					t.Fatalf("Response Bytes from JetStream do not match. Actual: %s. Expected: %s", string(resBytes), expected)
				}
			},
		},
	}

	// we share the same NATS Server and Caddy Server for all testcases
	_, nc := StartTestNats(t)
	caddyTester := caddytest.NewTester(t)

	for _, testcase := range cases {
		t.Run(testcase.description, func(t *testing.T) {

			subscription, err := nc.SubscribeSync("greet.>")
			defer subscription.Unsubscribe()
			failOnErr("error subscribing to greet.>: %w", err, t)

			caddyTester.InitServer(defaultCaddyConf+`
				:8889 {
					route /test/* {
						nats_publish greet.hello
					}
				}
			`, "caddyfile")

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
