package integration

import (
	"fmt"
	"github.com/caddyserver/caddy/v2/caddytest"
	"github.com/nats-io/nats.go"
	"io"
	"net/http"
	_ "sandstorm.de/custom-caddy/nats-bridge"
	"testing"
	"time"
)

// TestRequestToNats converts a HTTP request to a NATS Publication, and vice-versa
// for the response.
//
//		              ┌──────────────┐    HTTP: /test
//		◀─────────────│ Caddy /test  │◀───────
//		NATS subject  │ nats_publish │
//		 greet.*      │              │
//	    ────────────▶ └──────────────┘ ────────────▶
func TestRequestToNats(t *testing.T) {
	type testCase struct {
		description                      string
		sendHttpRequestAndAssertResponse func() error
		handleNatsMessage                func(msg *nats.Msg, nc *nats.Conn) error
		CaddyfileSnippet                 string
	}

	// Testcases
	cases := []testCase{
		{
			description: "Simple GET request should keep headers and contain extra X-NatsHttp-Method and X-NatsHttp-UrlPath",
			sendHttpRequestAndAssertResponse: func() error {
				// 1) send initial HTTP request (will be validated on the NATS handler side)
				req, err := http.NewRequest("GET", "http://localhost:8889/test/hi", nil)
				if err != nil {
					return err
				}
				req.Header.Add("Custom-Header", "MyValue")
				res, err := http.DefaultClient.Do(req)
				if err != nil {
					return fmt.Errorf("HTTP request failed: %w", err)
				}

				// 4) validate HTTP response
				b, err := io.ReadAll(res.Body)
				if err != nil {
					return fmt.Errorf("could not read response body: %w", err)
				}
				if string(b) != "respData" {
					return fmt.Errorf("wrong response body. Expected: respData. Actual: %s", string(b))
				}
				if actualH := res.Header.Get("RespHeader"); actualH != "RespHeaderValue" {
					return fmt.Errorf("wrong response header. Expected: RespHeaderValue. Actual: %s. Full Headers: %+v", actualH, res.Header)
				}

				return nil
			},
			CaddyfileSnippet: `
				route /test/* {
					nats_request greet.hello
				}
			`,
			handleNatsMessage: func(msg *nats.Msg, nc *nats.Conn) error {
				// 2) validate incoming NATS request (converted from HTTP)
				if msg.Header.Get("Custom-Header") != "MyValue" {
					t.Fatalf("Custom-Header not correct, expected 'MyValue', actual headers: %+v", msg.Header)
				}

				if msg.Header.Get("X-NatsHttp-Method") != "GET" {
					t.Fatalf("X-NatsHttp-Method not correct, expected 'GET', actual headers: %+v", msg.Header)
				}
				if msg.Header.Get("X-NatsHttp-UrlPath") != "/test/hi" {
					t.Fatalf("X-NatsHttp-UrlPath not correct, expected '/test/hi', actual headers: %+v", msg.Header)
				}
				if msg.Header.Get("X-NatsHttp-UrlQuery") != "" {
					t.Fatalf("X-NatsHttp-UrlQuery not correct, expected '', actual headers: %+v", msg.Header)
				}

				// 3) send NATS response (will be validated on the HTTP response side)
				resp := &nats.Msg{
					Data:   []byte("respData"),
					Header: make(nats.Header),
				}
				resp.Header.Add("RespHeader", "RespHeaderValue")
				return msg.RespondMsg(resp)
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

			// HTTP Request and assertion Goroutine
			httpResultChan := make(chan error)
			go func() {
				httpResultChan <- testcase.sendHttpRequestAndAssertResponse()
			}()

			// handle NATS message and generate response.
			msg, err := subscription.NextMsg(10 * time.Millisecond)
			if err != nil {
				t.Fatalf("message not received: %v", err)
			} else {
				t.Logf("Received message: %+v", msg)
			}
			err = testcase.handleNatsMessage(msg, nc)
			if err != nil {
				t.Fatalf("error with NATS message: %s", err)
			}

			// now, wait until the HTTP request goroutine finishes (and did its assertions)
			httpResult := <-httpResultChan
			if httpResult != nil {
				t.Fatalf("error with HTTP Response message: %s", err)
			}
		})
	}
}
