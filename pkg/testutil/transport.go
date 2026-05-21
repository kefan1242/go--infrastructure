package testutil

import (
	"context"

	"github.com/go-kratos/kratos/v2/transport"
)

// FakeTransport implements kratos transport.Transporter for middleware tests.
// kind and op are customizable; header maps are exposed for read/write.
type FakeTransport struct {
	kind      transport.Kind
	op        string
	reqHeader transport.Header
	rspHeader transport.Header
}

type fakeHeader map[string]string

func (h fakeHeader) Get(key string) string { return h[key] }
func (h fakeHeader) Set(key, value string) { h[key] = value }
func (h fakeHeader) Add(key, value string) { h[key] = value }
func (h fakeHeader) Keys() []string {
	out := make([]string, 0, len(h))
	for k := range h {
		out = append(out, k)
	}
	return out
}

func (h fakeHeader) Values(key string) []string {
	if v, ok := h[key]; ok {
		return []string{v}
	}
	return nil
}

// NewFakeTransport returns a transport with sensible defaults (kind=gRPC,
// op=/test.Test/Test, empty headers).
func NewFakeTransport(opts ...FakeTransportOption) *FakeTransport {
	t := &FakeTransport{
		kind:      transport.KindGRPC,
		op:        "/test.Test/Test",
		reqHeader: fakeHeader{},
		rspHeader: fakeHeader{},
	}
	for _, o := range opts {
		o(t)
	}
	return t
}

// FakeTransportOption is a functional option for NewFakeTransport.
type FakeTransportOption func(*FakeTransport)

func WithKind(k transport.Kind) FakeTransportOption { return func(t *FakeTransport) { t.kind = k } }
func WithOp(op string) FakeTransportOption          { return func(t *FakeTransport) { t.op = op } }
func WithReqHeader(k, v string) FakeTransportOption {
	return func(t *FakeTransport) { t.reqHeader.Set(k, v) }
}

// transport.Transporter implementation.
func (t *FakeTransport) Kind() transport.Kind            { return t.kind }
func (t *FakeTransport) Endpoint() string                { return "fake://test" }
func (t *FakeTransport) Operation() string               { return t.op }
func (t *FakeTransport) RequestHeader() transport.Header { return t.reqHeader }
func (t *FakeTransport) ReplyHeader() transport.Header   { return t.rspHeader }

// InjectServerContext attaches the transport to ctx so transport.FromServerContext
// can read it back. Use for testing server-side middlewares.
func InjectServerContext(ctx context.Context, t *FakeTransport) context.Context {
	return transport.NewServerContext(ctx, t)
}

// InjectClientContext is the client-side counterpart.
func InjectClientContext(ctx context.Context, t *FakeTransport) context.Context {
	return transport.NewClientContext(ctx, t)
}
