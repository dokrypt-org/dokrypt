package evm

import (
	"context"
	"encoding/json"

	"github.com/dokrypt/dokrypt/internal/rpc"
)

type RPCClient struct {
	inner *rpc.Client
}

func NewRPCClient(url string) *RPCClient {
	return &RPCClient{
		inner: rpc.NewClient(url),
	}
}

func (c *RPCClient) Call(ctx context.Context, method string, params ...any) (json.RawMessage, error) {
	return c.inner.Call(ctx, method, params...)
}

func (c *RPCClient) CallResult(ctx context.Context, v any, method string, params ...any) error {
	return c.inner.CallResult(ctx, v, method, params...)
}

func (c *RPCClient) BatchCall(ctx context.Context, calls []rpc.BatchRequest) ([]rpc.Response, error) {
	return c.inner.BatchCall(ctx, calls)
}

func (c *RPCClient) SetURL(url string) {
	c.inner.SetURL(url)
}

func (c *RPCClient) URL() string {
	return c.inner.URL()
}

func (c *RPCClient) Inner() *rpc.Client {
	return c.inner
}
