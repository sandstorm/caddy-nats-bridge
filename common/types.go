package common

import "github.com/nats-io/nats.go"

type NatsHandler interface {
	Subscribe(conn *nats.Conn) error
	Unsubscribe(conn *nats.Conn) error
}
