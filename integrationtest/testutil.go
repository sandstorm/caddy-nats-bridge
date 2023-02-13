package integrationtest

import "testing"

func FailOnErr(message string, err error, t *testing.T) {
	t.Helper()
	if err != nil {
		t.Fatalf(message, err)
	}
}

const DefaultCaddyConf = `
{
	default_bind 127.0.0.1
	http_port 8889
	admin 127.0.0.1:2999
	nats {
		url 127.0.0.1:8369
		%s
	}
}
`
