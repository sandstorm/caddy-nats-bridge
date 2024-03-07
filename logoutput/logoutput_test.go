package logoutput_test

import (
	"encoding/json"
	"github.com/nats-io/nats.go"
)

import (
	"fmt"
	_ "github.com/sandstorm/caddy-nats-bridge"
	"github.com/sandstorm/caddy-nats-bridge/integrationtest"
	"net/http"
	"testing"
	"time"
)

// example log message:
// {"level":"info","ts":1709835557.872107,"logger":"http.log.access.log0","msg":"handled request","request":{"remote_ip":"127.0.0.1","remote_port":"50174","client_ip":"127.0.0.1","proto":"HTTP/1.1","method":"GET","host":"localhost:8889","uri":"/test/hi","headers":{"User-Agent":["Go-http-client/1.1"],"Accept-Encoding":["gzip"]}},"bytes_read":0,"user_id":"","duration":0.000015458,"size":0,"status":200,"resp_headers":{"Server":["Caddy"],"Content-Type":[]}}
type logMsgExample struct {
	Level   string  `json:"level"`
	Ts      float64 `json:"ts"`
	Logger  string  `json:"logger"`
	Msg     string  `json:"msg"`
	Request struct {
		RemoteIp   string `json:"remote_ip"`
		RemotePort string `json:"remote_port"`
		ClientIp   string `json:"client_ip"`
		Proto      string `json:"proto"`
		Method     string `json:"method"`
		Host       string `json:"host"`
		Uri        string `json:"uri"`
	} `json:"request"`
	BytesRead int     `json:"bytes_read"`
	UserId    string  `json:"user_id"`
	Duration  float64 `json:"duration"`
	Size      int     `json:"size"`
	Status    int     `json:"status"`
}

func TestLogRequestToNats(t *testing.T) {
	type testCase struct {
		description                      string
		sendHttpRequestAndAssertResponse func() error
		handleNatsMessage                func(msg *nats.Msg, nc *nats.Conn) error
		CaddyfileSnippet                 string
	}

	// Testcases
	cases := []testCase{
		{
			description: "HTTP request logging to NATS",
			sendHttpRequestAndAssertResponse: func() error {
				// 1) send initial HTTP request (will be validated on the NATS handler side)
				req, err := http.NewRequest("GET", "http://localhost:8889/test/hi", nil)
				if err != nil {
					return err
				}
				_, err = http.DefaultClient.Do(req)
				if err != nil {
					return fmt.Errorf("HTTP request failed: %w", err)
				}

				return nil
			},
			CaddyfileSnippet: `
				log {
					output nats my.subject
				}
				route /test/* {
					respond 200
				}
			`,
			handleNatsMessage: func(msg *nats.Msg, nc *nats.Conn) error {
				// 2) validate incoming NATS request (converted from HTTP)
				if msg.Subject != "my.subject" {
					t.Fatalf("Subject not correct, expected 'my.subject', actual: %s", msg.Subject)
				}
				// {"level":"info","ts":1709835557.872107,"logger":"http.log.access.log0","msg":"handled request","request":{"remote_ip":"127.0.0.1","remote_port":"50174","client_ip":"127.0.0.1","proto":"HTTP/1.1","method":"GET","host":"localhost:8889","uri":"/test/hi","headers":{"User-Agent":["Go-http-client/1.1"],"Accept-Encoding":["gzip"]}},"bytes_read":0,"user_id":"","duration":0.000015458,"size":0,"status":200,"resp_headers":{"Server":["Caddy"],"Content-Type":[]}}
				var logMsg logMsgExample
				err := json.Unmarshal(msg.Data, &logMsg)
				if err != nil {
					return err
				}

				if logMsg.Level != "info" {
					t.Fatalf("msg.level not correct, actual: %s", logMsg.Level)
				}
				if logMsg.Msg != "handled request" {
					t.Fatalf("msg.msg not correct, actual: %s", logMsg.Msg)
				}
				if logMsg.Status != 200 {
					t.Fatalf("msg.status not correct, actual: %d", logMsg.Status)
				}

				return nil
			},
		},
	}

	// we share the same NATS Server and Caddy Server for all testcases
	_, nc := integrationtest.StartTestNats(t)
	caddyTester := integrationtest.NewCaddyTester(t)

	for _, testcase := range cases {
		t.Run(testcase.description, func(t *testing.T) {

			subscription, err := nc.SubscribeSync(">")
			defer subscription.Unsubscribe()
			integrationtest.FailOnErr("error subscribing to >: %w", err, t)

			caddyTester.InitServer(fmt.Sprintf(integrationtest.DefaultCaddyConf+`
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
