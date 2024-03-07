#!/usr/bin/env bash

# detect location of XCADDY
XCADDY=xcaddy
if [ -f ~/go/bin/xcaddy ]; then
  XCADDY=~/go/bin/xcaddy
fi

# build caddy server if needed
if [ ! -f caddy ]; then
  go get -u github.com/sandstorm/caddy-nats-bridge
  $XCADDY build --with github.com/sandstorm/caddy-nats-bridge
fi

nats-server &
sleep 1
trap 'killall nats-server' EXIT

./caddy run --config Caddyfile

