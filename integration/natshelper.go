package integration

import (
	"fmt"
	"github.com/nats-io/nats-server/v2/server"
	natsserver "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
	"testing"
)

const TEST_PORT = 8369

func StartTestNats(t *testing.T) (*server.Server, *nats.Conn) {
	natsServer := RunServerOnPort(TEST_PORT)
	t.Cleanup(func() {
		natsServer.Shutdown()
	})

	serverUrl := fmt.Sprintf("nats://127.0.0.1:%d", TEST_PORT)
	natsClient, err := nats.Connect(serverUrl)
	if err != nil {
		t.Fatalf("Nats client could not be created: %v", err)
	}
	t.Cleanup(func() {
		natsClient.Drain()
	})

	return natsServer, natsClient
}

func RunServerOnPort(port int) *server.Server {
	opts := natsserver.DefaultTestOptions
	opts.Port = port
	opts.JetStream = true
	opts.Debug = true
	opts.Trace = true
	opts.NoLog = false
	return RunServerWithOptions(&opts)
}

func RunServerWithOptions(opts *server.Options) *server.Server {
	return natsserver.RunServer(opts)
}
