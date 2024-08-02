# Caddy-NATS-Bridge

> NOTE: This package is originally based on https://github.com/codegangsta/caddy-nats,
> which has greatly inspired me to work on this topic. Because we want to use this
> package in our core infrastructure very heavily and I had some specific ideas
> around the API, I created my own package based on the original one.
> 
> tl;dr: Pick whichever works for you - Open Source rocks :)

`caddy-nats-bridge` is a caddy module that allows the caddy server to interact with a
[NATS](https://nats.io/) server in meaningful ways.

**Main Features**:

- Connect NATS and HTTP in all possible directions (we call this **Bridging**)
- experimental: offload large HTTP bodies to JetStream
- NEW in 0.7: publish Caddy Log messages to NATS

The initial of this project was to better bridge HTTP based services with NATS
in a pragmatic and straightforward way. If you've been wanting to use NATS, but
have some use cases that still need to use HTTP, this may be a really good
option for you. 

This extension supports multiple patterns:
publish/subscribe, fan in/out, and request reply.

Additionally, this extension supports using NATS as Log output.

<!-- TOC -->
* [Caddy-NATS-Bridge](#caddy-nats-bridge)
* [Installation](#installation)
* [Getting Started - NATS as Log Output](#getting-started---nats-as-log-output)
* [Getting Started - Bridging HTTP <-> NATS](#getting-started---bridging-http---nats)
* [Connecting to NATS](#connecting-to-nats)
* [Logging to NATS](#logging-to-nats)
* [Bridging HTTP <-> NATS](#bridging-http---nats)
  * [NATS -> HTTP via `subscribe`](#nats---http-via-subscribe)
    * [Placeholders for `subscribe`](#placeholders-for-subscribe)
    * [Queue Groups](#queue-groups)
    * [FAQ: HTTP URL Parameters](#faq-http-url-parameters)
  * [HTTP -> NATS via `nats_request` (interested about response)](#http---nats-via-nats_request-interested-about-response)
    * [Placeholders for `nats_request`](#placeholders-for-nats_request)
    * [Extra headers for `nats_request`](#extra-headers-for-nats_request)
  * [HTTP -> NATS via `nats_publish` (fire-and-forget)](#http---nats-via-nats_publish-fire-and-forget)
    * [Placeholders for `nats_publish`](#placeholders-for-nats_publish)
    * [Extra headers for `nats_publish`](#extra-headers-for-nats_publish)
  * [large HTTP payloads with store_body_to_jetstream](#large-http-payloads-with-store_body_to_jetstream)
  * [Development](#development)
<!-- TOC -->

# Installation

To use `caddy-nats-bridge`, simply run the [xcaddy](https://github.com/caddyserver/xcaddy) build tool to create a
`caddy-nats-bridge` compatible caddy server.

```sh
# Prerequisites - install go and xcaddy
brew install go
go install github.com/caddyserver/xcaddy/cmd/xcaddy@latest

# now, build your custom Caddy server (NOTE: path to xcaddy might differ on your system).
~/go/bin/xcaddy build --with github.com/sandstorm/caddy-nats-bridge
```

# Getting Started - NATS as Log Output

(supported since version 0.7.0 of this extension).

Getting up and running with `caddy-nats-bridge` is pretty simple:

First [install NATS](https://docs.nats.io/running-a-nats-service/introduction/installation) and make sure the NATS server is running:

```sh
nats-server
```

> :bulb: To try this example, `cd examples/logs; ./build-run.sh`
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
	}
}

http://127.0.0.1:8888 {
	log {
		output nats my.log.subject
	}
	respond "Hello World"
}
```

Then, you can can listen to `my.log` with the `nats` CLI, and do a HTTP Request to see it in the Logs:

```bash
nats subscribe 'my.log.>'

# in another console
curl http://127.0.0.1:8888
```

What has happened here?

1. You sent a HTTP request to Caddy
2. the access log is routed to NATS.


# Getting Started - Bridging HTTP <-> NATS

This is a more advanced scenario, but still getting up and running with `caddy-nats-bridge` is pretty simple:

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
		# nats req "datausa.Nation.Population" ""
		subscribe datausa.> GET http://127.0.0.1:8888/datausa/{nats.request.subject.asUriPath.1}/{nats.request.subject.asUriPath.2}
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

# Connecting to NATS

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

# Logging to NATS

Simple usage:

```
{
  nats {
    url nats://127.0.0.1:4222
  }
}

my-domain.com {
  # will use the "default" NATS server defined above
  log nats my.log.subject
}
```

You can also specify the NATS server alias to use for logging; in the example below "myNatsServer":

```
{
nats {
    url nats://127.0.0.1:4222
  }
  nats myNatsServer {
    url nats://127.0.0.1:5444
  }
}

my-domain.com {
  # will use the "myNatsServer" NATS server defined above
  log nats myNatsServer my.log.subject
}
```

This concept is fully pluggable; you can configure the log output any way you like in Caddy.

# Bridging HTTP <-> NATS

![](./connectivity-modes.drawio.png)

The module works if you want to bridge *HTTP -> NATS*, and also *NATS -> HTTP* - both in unidirectional, and in
bidirectional mode.


## NATS -> HTTP via `subscribe`

`caddy-nats-bridge` supports subscribing to NATS subjects in a few different flavors, depending on your needs:

- forward a NATS message to an HTTP endpoint, and replying the HTTP response back to the NATS sender.
- forward a NATS message to an HTTP endpoint, without interest for the response (f.e. triggering a webhook).

The two cases above are both handled with the single `subscribe` directive placed inside a global `nats` block.
If the incoming [NATS Message](https://pkg.go.dev/github.com/nats-io/nats.go#Msg) has a `Reply` subject set,
we forward the HTTP response back to the original sender. If not, we discard the HTTP response.

We route the message inside Caddy to the matching `server` block (depending on the hostname). Then you can use
further Caddy directives for processing the message.

NATS headers get converted to HTTP request headers. HTTP response headers are converted to NATS headers on
the reply message (if applicable).

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

### Placeholders for `subscribe`

The `subscribe` directive supports the [global placeholders of Caddy](https://caddyserver.com/docs/conventions#placeholders)
inside the `http_method` and `http_url` parameters. Additional, the following (NATS specific) placeholders can be used:

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

### FAQ: HTTP URL Parameters

in case you want to use request parameters, I suggest the following way of using `subscribe`:

```
subscribe datausa.> GET http://127.0.0.1:8081/{nats.subject.asUriPath.1:}?{nats.request.header.X-NatsBridge-UrlQuery}
```

This way, the NATS request header `X-NatsBridge-UrlQuery` can be used to set URL parameters.

---
## HTTP -> NATS via `nats_request` (interested about response)

```nginx
nats_request [matcher] [serverAlias] subject {
  [timeout 42ms]
  [headers true|false]
}
```

`nats_request` publishes the HTTP request to the specified NATS subject, and
sends the NATS reply back as HTTP response. It is a terminal handler,
meaning it does not make sense to place Caddy handlers after this one because
they will never be called.

HTTP request headers are converted to NATS headers if the headers subdirective is set to true. 
NATS reply headers are converted to HTTP response headers.

For `matcher`, all registered [Caddy request matchers](https://caddyserver.com/docs/json/apps/http/servers/routes/match/)
can be used - and the `nats_request` handler is only triggered if the request matches the matcher. 

If `serverAlias` is not given, `default` is used.

**Example usage:**

```nginx
localhost {
  route /hello {
    nats_request events.hello
  }
}
```

### Placeholders for `nats_request`

(same as for `nats_publish`)

You can use the [http placeholders of Caddy](https://caddyserver.com/docs/json/apps/http/#docs) and
the [global placeholders of Caddy](https://caddyserver.com/docs/conventions#placeholders) inside the
`subject` string. The most useful ones are usually:

```
{http.request.uri.path} 	The path component of the request URI
{http.request.uri.path.*} 	Parts of the path, split by / (0-based from left)
{http.request.header.*}     Specific request header field
```

Additionally, the following placeholders are available:

- `http.request.uri.path.asNatsSubject`: The URI path, with slashes `/` replaced with a dot `.` to make it easy to
  map it to a NATS subject.
- `{http.request.uri.path.asNatsSubject.*}`: You can also select a segment of a path by index (0-based):
  `{http.request.uri.path.asNatsSubject.0}`  or a subslice of a path: `{http.request.uri.path.asNatsSubject.2:}`
  or `{http.request.uri.path.asNatsSubject.2:7}` to have ultimate flexibility how to build the NATS subject.

  Examples:
  ```
  # incoming URL path: /project/sandstorm/events
  
  {http.request.uri.path.asNatsSubject.0} => project
  {http.request.uri.path.asNatsSubject.1} => sandstorm
  {http.request.uri.path.asNatsSubject.1:} => sandstorm.events
  {http.request.uri.path.asNatsSubject.0:1} => project.sandstorm
  ```

### Extra headers for `nats_request`

(same as for `nats_publish`)

If the subdirective 'headers' is set to true, then all HTTP headers will become NATS Message headers. On top if this, the following headers are automatically set:

- `X-NatsBridge-Method` header: contains the HTTP header `GET,POST,HEAD,...`
- `X-NatsBridge-UrlPath` header: URI path without query string
- `X-NatsBridge-UrlQuery` header: encoded query values, without `?`


---
## HTTP -> NATS via `nats_publish` (fire-and-forget)

```nginx
nats_publish [matcher] [serverAlias] subject {
  [headers true|false]
}
```

`nats_publish` publishes the HTTP request to the specified NATS subject. This
http handler is not a terminal handler, which means it can be used as
middleware (Think logging and events for specific http requests).

HTTP request headers are converted to NATS headers.

For `matcher`, all registered [Caddy request matchers](https://caddyserver.com/docs/json/apps/http/servers/routes/match/)
can be used - and the `nats_request` handler is only triggered if the request matches the matcher.

If `serverAlias` is not given, `default` is used.

**Example usage:**

```nginx
localhost {
  route /hello {
    nats_publish events.hello
    respond "Hello, world"
  }
}
```

### Placeholders for `nats_publish`

(same as for `nats_request`)

You can use the [http placeholders of Caddy](https://caddyserver.com/docs/json/apps/http/#docs) and
the [global placeholders of Caddy](https://caddyserver.com/docs/conventions#placeholders) inside the
`subject` string. The most useful ones are usually:

```
{http.request.uri.path} 	The path component of the request URI
{http.request.uri.path.*} 	Parts of the path, split by / (0-based from left)
{http.request.header.*}     Specific request header field
```

Additionally, the following placeholders are available:

- `http.request.uri.path.asNatsSubject`: The URI path, with slashes `/` replaced with a dot `.` to make it easy to
  map it to a NATS subject.
- `{http.request.uri.path.asNatsSubject.*}`: You can also select a segment of a path by index (0-based):
  `{http.request.uri.path.asNatsSubject.0}`  or a subslice of a path: `{http.request.uri.path.asNatsSubject.2:}`
  or `{http.request.uri.path.asNatsSubject.2:7}` to have ultimate flexibility how to build the NATS subject.

  Examples:
  ```
  # incoming URL path: /project/sandstorm/events
  
  {http.request.uri.path.asNatsSubject.0} => project
  {http.request.uri.path.asNatsSubject.1} => sandstorm
  {http.request.uri.path.asNatsSubject.1:} => sandstorm.events
  {http.request.uri.path.asNatsSubject.0:1} => project.sandstorm
  ```

### Extra headers for `nats_publish`

(same as for `nats_request`)

If the subdirective 'headers' is set to true, then all HTTP headers will become NATS Message headers. On top if this, the following headers are automatically set:

- `X-NatsBridge-Method` header: contains the HTTP header `GET,POST,HEAD,...`
- `X-NatsBridge-UrlPath` header: URI path without query string
- `X-NatsBridge-UrlQuery` header: encoded query values, without `?`


---
## large HTTP payloads with store_body_to_jetstream

`store_body_to_jetstream` is experimental and might change without further notice as this feature is developed further.

NATS messages have a size limit of usually 1 MB (and 8 MB as hardcoded limit). Sometimes, HTTP requests or responses
contain bigger payloads than this.

`store_body_to_jetstream` takes a HTTP request's body and stores it to a temporary [JetStream Object Storage (EXPERIMENTAL)](https://docs.nats.io/using-nats/developer/develop_jetstream/object).
It then adds the NATS message headers `X-NatsBridge-Body-Bucket` pointing to the JetStream Object Store bucket,
and `X-NatsBridge-Body-Id` pointing to the object ID.
It then empties the body of the HTTP message before it is processed further.

```nginx
store_body_to_jetstream [<matcher>] [[serverAlias] bucketName] {
   [ttl 5m]
}
```

For `matcher`, all registered [Caddy request matchers](https://caddyserver.com/docs/json/apps/http/servers/routes/match/)
can be used - and the `nats_request` handler is only triggered if the request matches the matcher.

If `serverAlias` is not given, `default` is used.

If `bucketName` is not given, `LargeHttpRequestBodies` is used. The bucket is auto-created using the specified `ttl`
if it does not exist.

`store_body_to_jetstream` must be placed *before* `nats_publish` or `nats_request` in order to do its work.

**Example usage:**

```nginx
localhost {
  route /hello {
    store_body_to_jetstream // TODO: condition if message is large or chunked.
    nats_publish events.hello
    respond "Hello, world"
  }
}
```

When sending a NATS message after store_body_to_jetstream, the following headers are set:

- `X-NatsBridge-Body-Bucket` header: pointing to the JetStream Object Store bucket
- `X-NatsBridge-Body-Id` header: pointing to the object ID

> This feature is, as already stated, **considered experimental**.
>
> We have the following development ideas around this:
> - We do not want to store all request bodies; but ideally only those only bigger than a size limit.
> - Additionally, we need to decide what to do with requests without a `Content-Length` - when using
>   `Transfer-Encoding: chunked`. Either we buffer these fully into memory (so that we know its size),
>   or we stream them directly to JetStream as they come in.
> - We need to create the "reverse" operation as well: take a HTTP response with the `X-NatsBridge-Body-Bucket`
>   and `X-NatsBridge-Body-Id` headers, and fetch the body from JetStream.
> - We need the same pair of operations for *upstream* HTTP responses.
>   - Maybe we should support re-using a response body based on cache etags?


## Development

All features have tests written. To run them, use `./dev.sh run-tests` - or use https://github.com/sandstorm/dev-script-runner
to run them.

**Long-Term Vision / Feature Ideas**

This tooling might be very useful for exposing HTTP based services in NATS.

Especially combined with other NATS features such as authorization, an implementer
could assume that NATS requests *are* already authenticated; and automatically
add outgoing API keys (when talking to public APIs). This should greatly simplify
API key management.

In the long run, we might be able to replace things like a Kubernetes Ingress completely,
and use caddy-nats-bridge in place of the Ingress - and then do the routing based on NATS
subjects. This might also greatly simplify quite some features like scale-to-zero deployments.

This might especially make sense together with Caddy extensions like [FrankenPHP](https://github.com/dunglas/frankenphp),
so we could generate replies without even needing to send out a HTTP message again.

Further feature ideas:

- Publish Caddy Request Logs to NATS
- use NATS KV as storage for Caddy (e.g. for certificates)
- register services for endpoints

**Thanks**

Thanks to Derek Collison and the NATS team for building such a great project, and same to
Matt Holt and the Caddy team. A special thanks to the .tech podcast team, because through
[this podcast](https://techpodcast.form3.tech/episodes/ep-5-tech-nats-with-founder-derek-collison)
I re-discovered NATS and I started to grasp what we could use it for.

A huge thanks to https://github.com/codegangsta/caddy-nats, from whose code-base I started, as I would
not have been able to build all this from scratch without a working implementation already.

And finally a big Thank You also to my team at https://sandstorm.de/, who are always encouraging
and receptive all the time I come along with a crazy idea.
