package integration

import (
	"bytes"
	"fmt"
	"github.com/caddyserver/caddy/v2/caddytest"
	"github.com/nats-io/nats.go"
	"net/http"
	_ "sandstorm.de/custom-caddy/nats-bridge"
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
				req, err := http.NewRequest("POST", "http://localhost:8889/test/hi", bytes.NewReader(body))
				failOnErr("Error creating request: %w", err, t)

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
				bucket := msg.Header.Get("X-NatsHttp-Body-Bucket")
				if len(bucket) == 0 {
					t.Fatalf("X-NatsHttp-Body-Bucket not set or empty.")
				}
				id := msg.Header.Get("X-NatsHttp-Body-Id")
				if len(id) == 0 {
					t.Fatalf("X-NatsHttp-Body-Id not set or empty.")
				}
				// Read from X-Large-Body-Id
				js, err := nc.JetStream()
				failOnErr("Error getting JetStream Client: %s", err, t)
				os, err := js.ObjectStore(bucket)
				failOnErr("Error getting ObjectStore "+bucket+": %s", err, t)
				resBytes, err := os.GetBytes(id)
				failOnErr("Error getting Key from ObjectStore: %s", err, t)

				expected := "my request body"

				if string(resBytes) != expected {
					t.Fatalf("Response Bytes from JetStream do not match. Actual: %s. Expected: %s", string(resBytes), expected)
				}
			},
		},
		{
			description: "for requests without body, no headers shoul dbe added.",
			buildHttpRequest: func(t *testing.T) *http.Request {
				req, err := http.NewRequest("GET", "http://localhost:8889/test/hi", nil)
				failOnErr("Error creating request: %w", err, t)
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
				bucket := msg.Header.Get("X-NatsHttp-Body-Bucket")
				if len(bucket) != 0 {
					t.Fatalf("X-NatsHttp-Body-Bucket is set, but should be empty.")
				}
				id := msg.Header.Get("X-NatsHttp-Body-Id")
				if len(id) != 0 {
					t.Fatalf("X-NatsHttp-Body-Id is set, but should be empty.")
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

			caddyTester.InitServer(fmt.Sprintf(defaultCaddyConf+`
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
			testcase.assertNatsMessage(msg, nc, t)
		})
	}
}
