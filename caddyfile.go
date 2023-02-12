package caddynats

func init() {

	//httpcaddyfile.RegisterHandlerDirective("nats_request", parseRequestHandler)
}

/*func parseRequestHandler(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	var p = Publish{
		WithReply: true,
		Timeout:   publishDefaultTimeout,
	}
	err := p.UnmarshalCaddyfile(h.Dispenser)
	return p, err
}


/*func parseQueueSubscribeHandler(d *caddyfile.Dispenser) (Subscribe, error) {
	s := Subscribe{}
	// TODO: handle errors better here
	if !d.AllArgs(&s.Subject, &s.QueueGroup, &s.Method, &s.URL) {
		return s, d.Err("wrong number of arguments")
	}

	return s, nil
}*/
