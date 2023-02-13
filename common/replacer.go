package common

import (
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/caddyserver/caddy/v2"
	"github.com/nats-io/nats.go"
)

func AddNATSPublishVarsToReplacer(repl *caddy.Replacer, req *http.Request) {
	natsVars := func(key string) (any, bool) {
		if req != nil {
			switch key {
			// generated nats subject
			case "http.request.uri.path.asNatsSubject":
				p := strings.Trim(req.URL.Path, "/")
				return strings.ReplaceAll(p, "/", "."), true
			}

			// subject parts
			if strings.HasPrefix(key, natsSubjectReplPrefix) {
				idxStr := key[len(natsSubjectReplPrefix):]
				p := strings.Trim(req.URL.Path, "/")
				parts := strings.Split(p, "/")
				s, ok := subSlice(parts, idxStr)

				return strings.Join(s, "."), ok
			}

		}

		return nil, false
	}

	repl.Map(natsVars)
}

func AddNatsSubscribeVarsToReplacer(repl *caddy.Replacer, msg *nats.Msg) {
	natsVars := func(key string) (any, bool) {
		if msg != nil {
			switch key {
			case "nats.request.subject":
				return msg.Subject, true
			// generated nats path
			case "nats.request.subject.asUriPath":
				return strings.ReplaceAll(msg.Subject, ".", "/"), true
			}

			if prefix := "nats.request.subject.asUriPath."; strings.HasPrefix(key, prefix) {
				idxStr := key[len(prefix):]
				parts := strings.Split(msg.Subject, ".")
				s, ok := subSlice(parts, idxStr)

				return strings.Join(s, "/"), ok
			}

			// subject parts
			if prefix := "nats.request.subject."; strings.HasPrefix(key, prefix) {
				idxStr := key[len(prefix):]
				parts := strings.Split(msg.Subject, ".")
				s, ok := subSlice(parts, idxStr)

				return strings.Join(s, "."), ok
			}

			if prefix := "nats.request.header."; strings.HasPrefix(key, prefix) {
				headerName := key[len(prefix):]
				return msg.Header.Get(headerName), true
			}
		}

		return nil, false
	}

	repl.Map(natsVars)
}

// subSlice returns a subslice of the given slice based off the string exp.
// expressions can be in the format of ":" "n", "n:", ":n", or "n:n", with n
// being a valid integer
func subSlice(s []string, exp string) ([]string, bool) {
	var a, b int
	var err error

	aStr, bStr, isRange := strings.Cut(exp, ":")

	if aStr == "" {
		a = 0
	} else {
		a, err = strconv.Atoi(aStr)
		if err != nil {
			return s, false
		}
	}

	if bStr == "" {
		b = len(s)
	} else {
		b, err = strconv.Atoi(bStr)
		if err != nil {
			return s, false
		}
	}

	outOfBounds := func(i int) bool {
		return i < 0 || i > len(s)+1
	}

	if outOfBounds(a) || outOfBounds(b) {
		return s, false
	}

	b = minMax(b, 0, len(s))
	a = minMax(a, 0, len(s))

	if isRange {
		return s[a:b], true
	} else {
		return s[a : a+1], true
	}
}

func minMax(i int, min int, max int) int {
	return int(math.Min(float64(max), math.Max(float64(min), float64(i))))
}

var natsSubjectReplPrefix = "http.request.uri.path.asNatsSubject."
