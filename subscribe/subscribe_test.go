package subscribe_test

import (
	"fmt"
	"github.com/nats-io/nats.go"
	_ "github.com/sandstorm/caddy-nats-bridge"
	"github.com/sandstorm/caddy-nats-bridge/integrationtest"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestSubscribeRequestToNats converts a NATS message to a HTTP request.
// depending on whether a NATS response subject is known, it will handle the response or not.
//
//		      	          ┌──────────────┐    HTTP Request: /test
//		─────────────────▶│ Caddy /test  │ ─────────────────▶
//		NATS subscription │ nats_publish │ ◀─────── Resp
//		 greet.*          └──────────────┘
//	 optional resp◀───────
func TestSubscribeRequestToNats(t *testing.T) {
	type testCase struct {
		description                string
		GlobalNatsCaddyfileSnippet string
		sendNatsRequest            func(nc *nats.Conn) error
		handleHttp                 func(w http.ResponseWriter, r *http.Request) error
		CaddyfileSnippet           func(svr *httptest.Server) string
	}

	// Testcases
	cases := []testCase{
		{
			description: "publish with payload, discarding response",
			sendNatsRequest: func(nc *nats.Conn) error {
				// 1) send initial NATS request (will be validated on the HTTP handler side)
				msg := &nats.Msg{
					Subject: "foo",
					Reply:   "",
					Header:  nats.Header{},
					Data:    []byte("paylod"),
				}
				msg.Header.Add("MyHeader", "myHeaderValue")
				return nc.PublishMsg(msg)
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
				// 2) validate incoming HTTP request (converted from NATS)
				if r.URL.Path != "/test/something" {
					return fmt.Errorf("URL Path does not match. Expected: /test/something. Actual: %s", r.URL.Path)
				}
				if hdr := r.Header.Get("MyHeader"); hdr != "myHeaderValue" {
					return fmt.Errorf("MyHeader does not match. Expected: myHeaderValue. Actual: %s", hdr)
				}
				b, err := io.ReadAll(r.Body)
				if err != nil {
					return err
				}
				if string(b) != "paylod" {
					return fmt.Errorf("body payload does not match. Expected: paylod. Actual: %s", string(b))
				}

				_, _ = w.Write([]byte(""))
				return nil
			},
		},
		{
			description: "request with payload, interested in response",
			sendNatsRequest: func(nc *nats.Conn) error {
				// 1) send initial NATS request (will be validated on the HTTP handler side)
				msg := &nats.Msg{
					Subject: "foo",
					Header:  nats.Header{},
					Data:    []byte("paylod"),
				}
				msg.Header.Add("MyHeader", "myHeaderValue")
				resp, err := nc.RequestMsg(msg, 1*time.Second)
				if err != nil {
					return err
				}
				// 4) validate NATS response
				actual := string(resp.Data)
				if actual != "resp" {
					return fmt.Errorf("response payload does not match expected. Actual: %s", actual)
				}
				actualH := resp.Header.Get("Resp-Header")
				if actualH != "RespHeaderValue" {
					return fmt.Errorf("response header payload does not match expected. Actual Resp Headers: %+v", resp.Header)
				}
				return nil
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
				// 2) validate incoming HTTP request (converted from NATS)
				if r.URL.Path != "/test/something" {
					return fmt.Errorf("URL Path does not match. Expected: /test/something. Actual: %s", r.URL.Path)
				}
				if hdr := r.Header.Get("MyHeader"); hdr != "myHeaderValue" {
					return fmt.Errorf("MyHeader does not match. Expected: myHeaderValue. Actual: %s", hdr)
				}
				b, err := io.ReadAll(r.Body)
				if err != nil {
					return err
				}
				if string(b) != "paylod" {
					return fmt.Errorf("body payload does not match. Expected: paylod. Actual: %s", string(b))
				}

				// 3) send HTTP response (will be validated on the NATS response side)
				w.Header().Add("Resp-Header", "RespHeaderValue")
				_, _ = w.Write([]byte("resp"))
				return nil
			},
		},
		{
			// with queue group, we simply check that the request comes through even if a queue group is configured.
			// we cannot easily test queue group behavior, because we would need to spin up two Caddy instances for this
			// and ensure the message only appears once.
			// the test is the same as "publish with payload, discarding response"
			description: "publish with payload, on queuegroup",
			sendNatsRequest: func(nc *nats.Conn) error {
				// 1) send initial NATS request (will be validated on the HTTP handler side)
				msg := &nats.Msg{
					Subject: "foo",
					Reply:   "",
					Header:  nats.Header{},
					Data:    []byte("paylod"),
				}
				msg.Header.Add("MyHeader", "myHeaderValue")
				return nc.PublishMsg(msg)
			},
			GlobalNatsCaddyfileSnippet: `
				subscribe foo POST http://localhost:8889/test/something {
					queue q
				}
			`,
			CaddyfileSnippet: func(svr *httptest.Server) string {
				return fmt.Sprintf(`
					route /test/* {
						reverse_proxy %s
					}
				`, svr.URL)
			},
			handleHttp: func(w http.ResponseWriter, r *http.Request) error {
				// 2) validate incoming HTTP request (converted from NATS)
				if r.URL.Path != "/test/something" {
					return fmt.Errorf("URL Path does not match. Expected: /test/something. Actual: %s", r.URL.Path)
				}
				if hdr := r.Header.Get("MyHeader"); hdr != "myHeaderValue" {
					return fmt.Errorf("MyHeader does not match. Expected: myHeaderValue. Actual: %s", hdr)
				}
				b, err := io.ReadAll(r.Body)
				if err != nil {
					return err
				}
				if string(b) != "paylod" {
					return fmt.Errorf("body payload does not match. Expected: paylod. Actual: %s", string(b))
				}

				_, _ = w.Write([]byte(""))
				return nil
			},
		},
		// WILDCARDS!!
	}

	// we share the same NATS Server and Caddy Server for all testcases
	_, nc := integrationtest.StartTestNats(t)
	caddyTester := integrationtest.NewCaddyTester(t)

	for _, testcase := range cases {
		t.Run(testcase.description, func(t *testing.T) {
			// start the test HTTP server
			httpResultChan := make(chan error)
			svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				err := testcase.handleHttp(w, r)
				httpResultChan <- err
				if err != nil {
					w.WriteHeader(500)
				}
			}))
			t.Cleanup(svr.Close)

			caddyTester.InitServer(fmt.Sprintf(integrationtest.DefaultCaddyConf+`
				:8889 {
					log
					%s
				}
			`, testcase.GlobalNatsCaddyfileSnippet, testcase.CaddyfileSnippet(svr)), "caddyfile")

			// send the actual NATS request
			natsResultChan := make(chan error)
			go func() {
				natsResultChan <- testcase.sendNatsRequest(nc)
			}()

			// wait until both NATS and HTTP goroutines are finished;
			// or any of them returns an error.
			wait := 2
			for {
				select {
				case err := <-natsResultChan:
					if err != nil {
						t.Fatalf("NATS error: %s", err)
					}
					wait--
				case err := <-httpResultChan:
					if err != nil {
						t.Fatalf("HTTP error: %s", err)
					}
					wait--
				}
				if wait == 0 {
					println("ALL DONE")
					return
				}
			}
		})
	}
}
