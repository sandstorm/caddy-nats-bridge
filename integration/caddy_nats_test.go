package integration

import (
	"github.com/caddyserver/caddy/v2/caddytest"
	"testing"
	"time"
)

// TestNats tries to experiment with NATS request / reply in a testcase.
func TestNats(t *testing.T) {
	_, nc := StartTestNats(t)

	sub, _ := nc.SubscribeSync("greet.*")
	nc.Publish("greet.joe", []byte("hello"))
	msg, err := sub.NextMsg(10 * time.Millisecond)

	if err != nil {
		t.Fatalf("message not received: %v", err)
	} else {
		t.Logf("Received message: %s", string(msg.Data))
	}
}

func TestCaddy(t *testing.T) {
	tester := caddytest.NewTester(t)
	tester.InitServer(`
		{
			http_port 8889
			admin localhost:2999
		}
		:8889 {
			respond "Hello"
		}
	`, "caddyfile")

	tester.AssertGetResponse("http://localhost:8889", 200, "Hello")
}
