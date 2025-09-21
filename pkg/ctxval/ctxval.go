package ctxval

import (
	"context"
	"sync"
)

func Wrap(ctx context.Context) context.Context {
	if _, ok := getClient(ctx); ok {
		// already wrapped
		return ctx
	}
	c := newClient(ctx)
	return context.WithValue(ctx, defKey, c)
}

func Set[K comparable, V any](ctx context.Context, k K, v V) {
	c, ok := getClient(ctx)
	if !ok {
		return
	}
	c.set(k, v)
}

func Get[K comparable, V any](ctx context.Context, k K) (V, bool) {
	c, ok := getClient(ctx)
	if !ok {
		return *new(V), false
	}
	v, ok := c.get(k).(V)
	return v, ok
}

type ctxKey struct{}

var defKey = ctxKey{}

type client struct {
	// as we don't expect to store a lot of values
	// so context is already enough
	storage context.Context

	// as we don't expect to have a lot of goroutines
	// so a simple mutex is enough
	m sync.Mutex
}

func (c *client) get(key any) any {
	c.m.Lock()
	defer c.m.Unlock()
	return c.storage.Value(key)
}

func (c *client) set(key any, value any) {
	c.m.Lock()
	defer c.m.Unlock()
	c.storage = context.WithValue(c.storage, key, value)
}

func getClient(ctx context.Context) (*client, bool) {
	c, ok := ctx.Value(defKey).(*client)
	return c, ok
}

// newClient creates a new client with the given context.
// then traps the context within the client.
func newClient(ctx context.Context) *client {
	c := &client{
		m:       sync.Mutex{},
		storage: ctx,
	}
	return c
}
