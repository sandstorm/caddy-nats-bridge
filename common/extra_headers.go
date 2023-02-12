package common

import "context"

type ExtraNatsMsgHeaders map[string]string

const ExtraNatsMsgHeadersKey = "ExtraNatsMsgHeaders"

func ExtraNatsMsgHeadersFromContext(ctx context.Context) ExtraNatsMsgHeaders {
	extraMsgHeaders, ok := ctx.Value(ExtraNatsMsgHeadersKey).(ExtraNatsMsgHeaders)
	if !ok {
		return make(ExtraNatsMsgHeaders)
	}
	return extraMsgHeaders
}

func (h ExtraNatsMsgHeaders) StoreInCtx(ctx context.Context) context.Context {
	return context.WithValue(ctx, ExtraNatsMsgHeadersKey, h)
}
