package integrationtest

import (
	"testing"
	"time"
)

// TestNats tries to experiment with NATS request / reply in a testcase.
func TestNatsExample(t *testing.T) {
	tn := StartTestNats(t)

	sub, _ := tn.ClientConn.SubscribeSync("greet.*")
	tn.ClientConn.Publish("greet.joe", []byte("hello"))
	msg, err := sub.NextMsg(10 * time.Millisecond)

	if err != nil {
		t.Fatalf("message not received: %v", err)
	} else {
		t.Logf("Received message: %s", string(msg.Data))
	}
}

// TestCaddy experiments with caddy server in a testcase
func TestCaddy(t *testing.T) {
	tester := NewCaddyTester(t)
	tester.InitServer(`
		{
			default_bind 127.0.0.1
			http_port 8889
			admin localhost:2999
		}
		127.0.0.1:8889 {
			respond "Hello"
		}
	`, "caddyfile")

	tester.AssertGetResponse("http://127.0.0.1:8889", 200, "Hello")
}
