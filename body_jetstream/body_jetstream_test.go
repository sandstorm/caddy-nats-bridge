package body_jetstream_test

import (
	"bytes"
	"fmt"
	"github.com/nats-io/nats.go"
	_ "github.com/sandstorm/caddy-nats-bridge"
	"github.com/sandstorm/caddy-nats-bridge/integrationtest"
	"net/http"
	"testing"
	"time"
)

// It does not expect a response.
//
//	              ┌──────────────┐   ┌──────────────┐    HTTP: /test
//	◀─────────────│ Caddy /test  │◀──│ store body   │◀───────
//	NATS subject  │ nats_publish │   │ to JetStream │
//	 greet.*      └──────────────┘   └──────────────┘
func TestPublishRequestToNatsWithBodyJetstream(t *testing.T) {
	type testCase struct {
		description                string
		buildHttpRequest           func(t *testing.T) *http.Request
		assertNatsMessage          func(msg *nats.Msg, nc *nats.Conn, t *testing.T)
		GlobalNatsCaddyfileSnippet string
		CaddyfileSnippet           string
	}

	// Testcases
	cases := []testCase{
		{
			description: "msg data should be empty and JetStream should contain HTTP Body",
			buildHttpRequest: func(t *testing.T) *http.Request {
				body := []byte("my request body")
				req, err := http.NewRequest("POST", "http://127.0.0.1:8889/test/hi", bytes.NewReader(body))
				integrationtest.FailOnErr("Error creating request: %w", err, t)

				req.Header.Add("Custom-Header", "MyValue")
				return req
			},
			GlobalNatsCaddyfileSnippet: ``,
			CaddyfileSnippet: `
				route /test/* {
					# order is important - store_body_to_jetstream must come first.
					store_body_to_jetstream
					nats_publish greet.hello
				}
			`,
			assertNatsMessage: func(msg *nats.Msg, nc *nats.Conn, t *testing.T) {
				if len(msg.Data) > 0 {
					t.Fatalf("Request Body should be empty. Actual data: %+v", string(msg.Data))
				}
				bucket := msg.Header.Get("X-NatsBridge-Body-Bucket")
				if len(bucket) == 0 {
					t.Fatalf("X-NatsBridge-Body-Bucket not set or empty.")
				}
				id := msg.Header.Get("X-NatsBridge-Body-Id")
				if len(id) == 0 {
					t.Fatalf("X-NatsBridge-Body-Id not set or empty.")
				}
				// Read from X-Large-Body-Id
				js, err := nc.JetStream()
				integrationtest.FailOnErr("Error getting JetStream ClientConn: %s", err, t)
				os, err := js.ObjectStore(bucket)
				integrationtest.FailOnErr("Error getting ObjectStore "+bucket+": %s", err, t)
				resBytes, err := os.GetBytes(id)
				integrationtest.FailOnErr("Error getting Key from ObjectStore: %s", err, t)

				expected := "my request body"

				if string(resBytes) != expected {
					t.Fatalf("Response Bytes from JetStream do not match. Actual: %s. Expected: %s", string(resBytes), expected)
				}
			},
		},
		{
			description: "for requests without body, no headers should be added.",
			buildHttpRequest: func(t *testing.T) *http.Request {
				req, err := http.NewRequest("GET", "http://127.0.0.1:8889/test/hi", nil)
				integrationtest.FailOnErr("Error creating request: %w", err, t)

				return req
			},
			GlobalNatsCaddyfileSnippet: ``,
			CaddyfileSnippet: `
				route /test/* {
					# order is important - store_body_to_jetstream must come first.
					store_body_to_jetstream
					nats_publish greet.hello
				}
			`,
			assertNatsMessage: func(msg *nats.Msg, nc *nats.Conn, t *testing.T) {
				if len(msg.Data) > 0 {
					t.Fatalf("Request Body should be empty. Actual data: %+v", string(msg.Data))
				}
				bucket := msg.Header.Get("X-NatsBridge-Body-Bucket")
				if len(bucket) != 0 {
					t.Fatalf("X-NatsBridge-Body-Bucket is set, but should be empty.")
				}
				id := msg.Header.Get("X-NatsBridge-Body-Id")
				if len(id) != 0 {
					t.Fatalf("X-NatsBridge-Body-Id is set, but should be empty.")
				}
			},
		},

		// TODO realistic case with Transfer-Encoding chunked:
		//// // NOTE: we need to use bufio.NewReader, to enforce a Transfer-Encoding=chunked. See net.http.NewRequestWithContext from Go Stdlib.
		//				req, err := http.NewRequest("POST", "http://127.0.0.1:8889/test/hi", bufio.NewReader(strings.NewReader("Small Request Body, but chunked transfer encoding")))
		//				req.Header.Add("Transfer-Encoding", "chunked")
	}

	// we share the same NATS Server and Caddy Server for all testcases
	tn := integrationtest.StartTestNats(t)
	caddyTester := integrationtest.NewCaddyTester(t)

	for _, testcase := range cases {
		t.Run(testcase.description, func(t *testing.T) {

			subscription, err := tn.ClientConn.SubscribeSync("greet.>")
			defer subscription.Unsubscribe()
			integrationtest.FailOnErr("error subscribing to greet.>: %w", err, t)

			caddyTester.InitServer(fmt.Sprintf(integrationtest.DefaultCaddyConf+`
				:8889 {
					%s
				}
			`, testcase.GlobalNatsCaddyfileSnippet, testcase.CaddyfileSnippet), "caddyfile")

			// Build request, and
			req := testcase.buildHttpRequest(t)
			caddyTester.AssertResponse(req, 200, "")
			msg, err := subscription.NextMsg(10 * time.Millisecond)

			if err != nil {
				t.Fatalf("message not received: %v", err)
			} else {
				t.Logf("Received message: %+v", msg)
			}
			testcase.assertNatsMessage(msg, tn.ClientConn, t)
		})
	}
}
