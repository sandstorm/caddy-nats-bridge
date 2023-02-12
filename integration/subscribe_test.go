package integration

import (
	"fmt"
	"github.com/caddyserver/caddy/v2/caddytest"
	"github.com/nats-io/nats.go"
	"net/http"
	"net/http/httptest"
	_ "sandstorm.de/custom-caddy/nats-bridge"
	"testing"
)

// TestSubscribeRequestToNats converts a NATS message to a HTTP request.
// depending on whether a NATS response subject is known, it will handle the response or not.
//
//       	          ┌──────────────┐    HTTP Request: /test
//	─────────────────▶│ Caddy /test  │ ─────────────────▶
//	NATS subscription │ nats_publish │X◀─────── Resp
//	 greet.*          └──────────────┘

func TestSubscribeRequestToNats(t *testing.T) {
	type testCase struct {
		description                string
		GlobalNatsCaddyfileSnippet string
		sendNatsRequest            func(nc *nats.Conn, t *testing.T)
		handleHttp                 func(w http.ResponseWriter, r *http.Request) error
		CaddyfileSnippet           func(svr *httptest.Server) string
	}

	// Testcases
	cases := []testCase{
		{
			description: "publish with payload",
			sendNatsRequest: func(nc *nats.Conn, t *testing.T) {
				msg := &nats.Msg{
					Subject: "foo",
					Reply:   "",
					Header:  nats.Header{},
					Data:    []byte("paylod"),
				}
				msg.Header.Add("MyHeader", "myHeaderValue")
				err := nc.PublishMsg(msg)
				failOnErr("Could not send request: %s", err, t)
			},
			GlobalNatsCaddyfileSnippet: `
				subscribe foo POST http://localhost:8889/test/something
			`,
			CaddyfileSnippet: func(svr *httptest.Server) string {
				return fmt.Sprintf(`
					route /test/* {
						reverse_proxy %s
					}
				`, svr.URL)
			},
			handleHttp: func(w http.ResponseWriter, r *http.Request) error {
				if r.URL.Path != "/test/something" {
					return fmt.Errorf("URL Path does not match. Expected: /test/something. Actual: %s", r.URL.Path)
				}
				if hdr := r.Header.Get("MyHeader"); hdr != "myHeaderValue" {
					return fmt.Errorf("MyHeader does not match. Expected: myHeaderValue. Actual: %s", hdr)
				}

				_, _ = w.Write([]byte(""))
				return nil
			},
		},
		// WILDCARDS!!
	}

	// we share the same NATS Server and Caddy Server for all testcases
	_, nc := StartTestNats(t)
	caddyTester := caddytest.NewTester(t)

	for _, testcase := range cases {
		t.Run(testcase.description, func(t *testing.T) {
			// start the test HTTP server
			errorChan := make(chan error)
			svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				err := testcase.handleHttp(w, r)
				errorChan <- err
				if err != nil {
					w.WriteHeader(500)
				}
			}))
			t.Cleanup(svr.Close)

			caddyTester.InitServer(fmt.Sprintf(defaultCaddyConf+`
				:8889 {
					log
					%s
				}
			`, testcase.GlobalNatsCaddyfileSnippet, testcase.CaddyfileSnippet(svr)), "caddyfile")

			testcase.sendNatsRequest(nc, t)

			err := <-errorChan
			if err != nil {
				t.Fatalf("%s", err)
			}
		})
	}
}
