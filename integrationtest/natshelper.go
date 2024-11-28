package integrationtest

import (
	"fmt"
	"github.com/nats-io/nats-server/v2/server"
	natsserver "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
	"testing"
)

const TEST_PORT = 8369

type TestNats struct {
	Server     *server.Server
	ClientConn *nats.Conn
}

func (tn *TestNats) RestartServer(t *testing.T) {
	t.Logf("Shutting down NATS Server")
	tn.Server.Shutdown()
	t.Logf("Starting NATS Server")
	tn.Server = runServerOnPort(TEST_PORT)
	t.Logf("Started NATS Server")
}

func StartTestNats(t *testing.T) TestNats {
	natsServer := runServerOnPort(TEST_PORT)

	serverUrl := fmt.Sprintf("nats://127.0.0.1:%d", TEST_PORT)
	natsClient, err := nats.Connect(serverUrl)
	if err != nil {
		t.Fatalf("Nats client could not be created: %v", err)
	}
	t.Cleanup(func() {
		natsClient.Drain()
	})

	tn := TestNats{
		Server:     natsServer,
		ClientConn: natsClient,
	}

	t.Cleanup(func() {
		tn.Server.Shutdown()
	})

	return tn
}

func runServerOnPort(port int) *server.Server {
	opts := natsserver.DefaultTestOptions
	opts.Port = port
	opts.JetStream = true
	opts.Debug = true
	opts.Trace = true
	opts.NoLog = false

	return natsserver.RunServer(&opts)
}
