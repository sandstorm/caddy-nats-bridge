package logoutput_test

import (
	"encoding/json"
	"github.com/caddyserver/caddy/v2/caddytest"
	"github.com/nats-io/nats.go"
	"io"
	"strings"
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
		description                          string
		sendHttpRequestAndAssertResponse     func(t *testing.T, tn *integrationtest.TestNats) error
		handleNatsMessage                    func(msg *nats.Msg, nc *nats.Conn) error
		CaddyfileSnippet                     string
		shouldReloadCaddyBeforeExecutingTest bool
	}

	// Testcases
	cases := []testCase{
		{
			description: "HTTP request logging to NATS",
			sendHttpRequestAndAssertResponse: func(t *testing.T, tn *integrationtest.TestNats) error {
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

		{
			description: "HTTP request logging to NATS should also work after reload of caddy",
			sendHttpRequestAndAssertResponse: func(t *testing.T, tn *integrationtest.TestNats) error {
				forceReloadCaddy(t)

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

		{
			description: "!!! UNSTABLE TEST !!! HTTP request logging to NATS should also work after restart of NATS",
			sendHttpRequestAndAssertResponse: func(t *testing.T, tn *integrationtest.TestNats) error {
				tn.RestartServer(t)
				// not nice ;)
				time.Sleep(1 * time.Second)

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
	testNats := integrationtest.StartTestNats(t)
	caddyTester := integrationtest.NewCaddyTester(t)

	for _, testcase := range cases {
		t.Run(testcase.description, func(t *testing.T) {

			subscription, err := testNats.ClientConn.SubscribeSync(">")
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
				httpResultChan <- testcase.sendHttpRequestAndAssertResponse(t, &testNats)
			}()

			// handle NATS message and generate response.
			msg, err := subscription.NextMsg(10000 * time.Millisecond)
			if err != nil {
				t.Fatalf("message not received: %v", err)
			} else {
				t.Logf("Received message: %+v", msg)
			}
			err = testcase.handleNatsMessage(msg, testNats.ClientConn)
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

func forceReloadCaddy(t *testing.T) {
	res, err := http.Get(fmt.Sprintf("http://localhost:%d/config/", caddytest.Default.AdminPort))
	if err != nil {
		t.Logf("Error reading config: %s", err)
		t.FailNow()
		return
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	client := &http.Client{
		Timeout: caddytest.Default.LoadRequestTimeout,
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("http://localhost:%d/load", caddytest.Default.AdminPort), strings.NewReader(string(body)))
	if err != nil {
		t.Logf("failed to create request: %s", err)
		t.FailNow()
		return
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Cache-Control", "must-revalidate")

	res, err = client.Do(req)
	if err != nil {
		t.Logf("unable to contact caddy server: %s", err)
		t.FailNow()
		return
	}
	return
}
