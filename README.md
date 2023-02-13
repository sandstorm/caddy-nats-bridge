# Caddy-NATS-Bridge

> NOTE: This package is originally based on https://github.com/codegangsta/caddy-nats,
> which has greatly inspired me to work on this topic. Because we want to use this
> package in our core infrastructure very heavily and I had some specific ideas
> around the API, I created my own package based on the original one.
> 
> tl;dr: Pick whichever works for you - Open Source rocks :)

`caddy-nats-bridge` is a caddy module that allows the caddy server to interact with a
[NATS](https://nats.io/) server. This extension supports multiple patterns:
publish/subscribe, fan in/out, and request reply.

The purpose of this project is to better bridge HTTP based services with NATS
in a pragmatic and straightforward way. If you've been wanting to use NATS, but
have some use cases that still need to use HTTP, this may be a really good
option for you.

<!-- TOC -->
* [Caddy-NATS-Bridge](#caddy-nats-bridge)
  * [Concept Overview](#concept-overview)
  * [Installation](#installation)
  * [Getting Started](#getting-started)
  * [Connecting to NATS](#connecting-to-nats)
  * [NATS -> HTTP via `subscribe`](#nats----http-via-subscribe)
    * [Subscribe Placeholders](#subscribe-placeholders)
    * [Queue Groups](#queue-groups)
  * [------------------------](#------------------------)
  * [Publishing to a NATS subject](#publishing-to-a-nats-subject)
    * [Publish Placeholders](#publish-placeholders)
    * [nats_publish](#natspublish)
      * [Syntax](#syntax)
      * [Example](#example)
      * [large HTTP payloads with store_body_to_jetstream](#large-http-payloads-with-storebodytojetstream)
    * [nats_request](#natsrequest)
      * [Syntax](#syntax-1)
      * [Example](#example-1)
      * [Format of the NATS message](#format-of-the-nats-message)
  * [Concept](#concept)
  * [What's Next?](#whats-next)
<!-- TOC -->

## Concept Overview

![](./connectivity-modes.drawio.png)

The module works if you want to bridge *HTTP -> NATS*, and also *NATS -> HTTP* - both in unidirectional, and in
bidirectional mode. 

## Installation

To use `caddy-nats-bridge`, simply run the [xcaddy](https://github.com/caddyserver/xcaddy) build tool to create a
`caddy-nats-bridge` compatible caddy server.

```sh
# Prerequisites - install go and xcaddy
brew install go
go install github.com/caddyserver/xcaddy/cmd/xcaddy@latest

# now, build your custom Caddy server (NOTE: path to xcaddy might differ on your system).
~/go/bin/xcaddy build --with github.com/sandstorm/caddy-nats-bridge
```

## Getting Started

Getting up and running with `caddy-nats-bridge` is pretty simple:

First [install NATS](https://docs.nats.io/running-a-nats-service/introduction/installation) and make sure the NATS server is running:

```sh
nats-server
```


> :bulb: To try this example, `cd examples/getting-started; ./build-run.sh`
> 
> - [Caddyfile](./examples/getting-started/Caddyfile)
> - [build-run.sh](./examples/getting-started/build-run.sh)

Then create and run your Caddyfile:

```nginx
# run with: ./caddy run --config Caddyfile

{
	nats {
		url 127.0.0.1:4222
		clientName "My Caddy Server"

		# listens to "datausa.[drilldowns].[measures]" -> calls internal URL
		subscribe datausa.> GET http://127.0.0.1:8888/datausa/{nats.subject.asUriPath.1}/{nats.subject.asUriPath.2}
	}
}

http://127.0.0.1:8888 {
	# internal URL: "/datausa/[drilldowns]/[measures]"
	handle /datausa/* {

		# Reference for placeholders: https://caddyserver.com/docs/json/apps/http/#docs
		rewrite * /api/data?drilldowns={http.request.uri.path.1}&measures={http.request.uri.path.2}

		reverse_proxy https://datausa.io {
			header_up Host {upstream_hostport}
		}
	}

	route /weather/* {
		nats_request cli.weather.{http.request.uri.path.1}
	}
}

```

Then, you can do a request with the `nats` CLI, and see that it is automatically converted to a HTTP Request:

```bash
nats req "datausa.Nation.Population" ""
```

What has happened here?

1. You sent a NATS request to the topic `datausa.Nation.Population`
2. the Caddy-Nats-Bridge has subscribed to the above topic, and
   routed the request to the internal URL `http://127.0.0.1:8888/datausa/Nation/Population`
3. The rest is standard Caddy configuration - triggering the API call
   `https://datausa.io/api/data?drilldowns=Nation&measures=Population`
4. The HTTP response is then converted to a NATS reply.

As a second example, you can start a fake NATS weather service using:

```bash
nats reply 'cli.weather.>' --command "curl -s wttr.in/{{2}}?format=3"
```

You can query this service via `nats request cli.weather.Dresden ""` to retrieve the current weather forecast.

Because of the `nats_request` rule in the `Caddyfile` above, you can also request
`http://127.0.0.1:8888/weather/Dresden` in your browser:
1. You sent a HTTP request to Caddy
2. This has been forwarded to the NATS topic `cli.weather.Dresden`, which is responded by the `nats reply` tool above.
3. The NATS response is converted to a HTTP response.

## Connecting to NATS

To connect to `nats`, use the `nats` global option in your Caddyfile with the URL of the NATS server:

```nginx
{
  nats [alias] {
    url nats://127.0.0.1:4222
  } 
}
```

The `alias` is a server-reference which is relevant if you want to connect to two NATS servers at the same time. If
omitted, the serverAlias `default` is used.

To connect to multiple servers (NATS Cluster), separate them with `,` in the `url` parameter.

The following connection options are supported for a server:

- `url`: URL(s) pointing to a NATS cluster. `nats://`, `tls://`, `ws://`  and `wss://` URLs are all supportted,
  if the NATS cluster supports them. multiple comma separated URLs can be specified, but they must be all pointing
  to the same NATS cluster.
- Authentication Options (only one of the ones below may be specified; in descending precedence):
  - `userCredentialFile` a [User Credential file](https://docs.nats.io/using-nats/developer/connecting/creds) as
    generated by `nsc` tool. You need this if you use the [Decentralized JWT Authentication/Authorization](https://docs.nats.io/running-a-nats-service/configuration/securing_nats/auth_intro/jwt)
    of NATS.
  - `nkeyCredentialFile` an [NKEY File](https://docs.nats.io/running-a-nats-service/configuration/securing_nats/auth_intro/nkey_auth)
    as generated by `nk -gen user -pubout`. You need this if you use NKEY Authentication of NATS.
  - (other authentication options are not supported right now, but we can add them if needed)
- `clientName` name of the NATS client - this is useful for monitoring.
- `inboxPrefix`. The "inbox" is the response subject for [request/reply](https://docs.nats.io/nats-concepts/core-nats/reqreply)
  messaging. Your server operator might tell you that you need to use a different inbox prefix than the default `_INBOX`
  for [security reasons](https://natsbyexample.com/examples/auth/private-inbox/cli).

Configuration with all configuration options is specified below:

```nginx
{
  nats [alias] {
    url nats://127.0.0.1:4222
    # either userCredentialFile or nkeyCredentialFile can be specified. If both are specified, userCredentialFile
    # takes precedence.
    userCredentialFile /path/to/file.creds
    nkeyCredentialFile /path/to/file.nk
    clientName MyClient
    inboxPrefix _INBOX_custom
  }
}
```

## NATS -> HTTP via `subscribe`

`caddy-nats-bridge` supports subscribing to NATS subjects in a few different flavors, depending on your needs:

- forward a NATS message to an HTTP endpoint, and replying the HTTP response back to the NATS sender.
- forward a NATS message to an HTTP endpoint, without interest for the response (f.e. triggering a webhook).

The two cases above are both handled with the single `subscribe` directive placed inside a global `nats` block.
If the incoming [NATS Message](https://pkg.go.dev/github.com/nats-io/nats.go#Msg) has a `Reply` subject set,
we forward the HTTP response back to the original sender. If not, we discard the HTTP response.

We route the message inside Caddy to the matching `server` block (depending on the hostname). Then you can use
further Caddy directives for processing the message.

```nginx
{
  nats [alias] {
    # add other server config here; at least URL is required.
    url nats://127.0.0.1:4222
    
    subscribe [topic] [http_method] [http_url] {
      [queue "queue group name"]
    }
    # example:
    subscribe datausa.> GET http://127.0.0.1:8081/{nats.subject.asUriPath.1:} 
  }
}

http://127.0.0.1:8081 {
  # your normal NATS config for the server.
}
```

### Subscribe Placeholders

The `subscribe` directive supports the following `caddy` placeholders in the `http_method` and `http_url` arguments:

- `{nats.request.subject}`: The subject of this message. 
  Example: `datausa.Nation.Population`
- `{nats.request.subject.asUriPath}`: The subject of this message, with dots "." replaced with a slash "/" to make it easy to map to a URL.
  Example: `datausa/Nation/Population`
- `{nats.request.subject.*}`: You can also select a segment of a path by index (0-based): `{nats.request.subject.0}`  or a subslice of a path: `{nats.request.subject.2:}` or `{nats.request.subject.2:7}` to have ultimate flexibility how to build the URL string.
  Examples:
  ```
  {nats.request.subject.0} => datausa
  {nats.request.subject.1} => Nation
  {nats.request.subject.1:} => Nation.Population
  {nats.request.subject.0:1} => datausa.Nation
  ```
- `{nats.request.subject.asUriPath.*}`: You can also select a segment of a path by index (0-based): `{nats.request.subject.asUriPath.0}`  or a subslice of a path: `{nats.request.subject.asUriPath.2:}` or `{nats.request.subject.asUriPath.2:7}` to have ultimate flexibility how to build the URL string.
  Examples:
  ```
  {nats.request.subject.asUriPath.0} => datausa
  {nats.request.subject.asUriPath.1} => Nation
  {nats.request.subject.asUriPath.1:} => Nation/Population
  {nats.request.subject.asUriPath.0:1} => datausa/Nation
  ```
- `{nats.request.header.*}`: output the given NATS message header.
  Example: `{nats.request.header.MyHeaderName}`

### Queue Groups

If you want to take part in Load Balancing via [NATS Queue Groups](https://docs.nats.io/nats-concepts/core-nats/queue),
you can specify the queue group to subscribe to via the nested `queue` directive inside the `subscribe` block.

------------------------
------------------------
------------------------

## Publishing to a NATS subject

`caddy-nats` also supports publishing to NATS subjects when an HTTP call is
matched within `caddy`, this makes for some very powerful bidirectional
patterns.

### Publish Placeholders

All `publish` based directives (`nats_publish`, `nats_request`) support the following `caddy` placeholders in the `subject` argument:

- `{nats.subject}`: The path of the http request, with slashes "/" replaced with dots "." to make it easy to map to a NATS subject.
- `{nats.subject.*}`: You can also select a segment of a subject ex: `{nats.subject.0}` or a subslice of the subject: `{nats.subject.2:}` or `{nats.subject.2:7}` to have ultimate flexibility in what to forward onto the URL path.

Additionally, since `publish` based directives are caddy http handlers, you also get access to all [caddy http placeholders](https://caddyserver.com/docs/modules/http#docs).

---

### nats_publish

#### Syntax
```nginx
nats_publish [<matcher>] <subject> {
  timeout <timeout-ms>
}
```
`nats_publish` publishes the request body to the specified NATS subject. This
http handler is not a terminal handler, which means it can be used as
middleware (Think logging and events for specific http requests).

#### Example

Publish an event before responding to the http request:

```nginx
localhost {
  route /hello {
    nats_publish events.hello
    respond "Hello, world"
  }
}
```

#### large HTTP payloads with store_body_to_jetstream

(TODO describe here)

```nginx
localhost {
  route /hello {
    store_body_to_jetstream // TODO: condition if message is large or chunked.
    nats_publish events.hello
    respond "Hello, world"
  }
}
```

---

### nats_request

#### Syntax
```nginx
nats_request [<matcher>] <subject> {
  timeout <timeout-ms>
}
```
`nats_request` publishes the request body to the specified NATS subject, and
writes the response of the NATS reply to the http response body.

#### Example

Publish an event before responding to the http request:

```nginx
localhost {
  route /hello/* {
    nats_request hello_service.{nats.subject.1}
  }
}
```

#### Format of the NATS message

- HTTP Body = NATS Message Data
- HTTP Headers = NATS Message Headers
  - `X-NatsBridge-Method` header: contains the HTTP header `GET,POST,HEAD,...`
  - `X-NatsBridge-UrlPath` header: URI path without query string
  - `X-NatsBridge-UrlQuery` header: query string
  - `X-NatsBridge-LargeBody-Bucket` header
  - `X-NatsBridge-LargeBody-Id` header
- NATS messages have a size limit of usually 1 MB (and 8 MB as hardcoded limit).
  In case the HTTP body is bigger, or alternatively, is submitted with `Transfer-Encoding: chunked` (so we do not know the size upfront);
  we do the following:
  - We store the HTTP body in the [JetStream Object Storage (EXPERIMENTAL)](https://docs.nats.io/using-nats/developer/develop_jetstream/object)
    for a few minutes; in a random key.
  - The name of this KV Storage key is stored in the `X-NatsBridge-LargeBody-Id`.
    - TODO: support for response body re-use based on cache etags?

## Large Body stuff

## Concept

- HTTP => NATS => HTTP should functionally emit the same requests (and responses)
- Big Request bodies should be stored to JetStream
  - for transfer encoding chunked
- (NATS -> HTTP) Big response bodies should be stored to JetStream
  - for transfer encoding chunked
  - for unknown response sizes
  - TODO: Re-use same cache entries if etags match?
- TODO: is request / response streaming necessary???
- All HTTP headers are passed through to NATS without modification
- All Nats headers except "X-NatsBridge-...." are passed to HTTP without modification
  - X-NatsBridge-Method
  - X-NatsBridge-JSBodyId
- allow multiple NATS servers

## What's Next?
While this is currently functional and useful as is, here are the things I'd like to add next:

- [ ] Add more examples in the /examples directory
- [ ] Add Validation for all caddy modules in this package
- [ ] Add godoc comments
- [ ] Support mapping nats headers and http headers for upstream and downstream
- [ ] Customizable error handling
- [ ] Jetstream support (for all that persistence babyyyy)
- [ ] nats KV for storage
