package exchange_test

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"testing"

	"github.com/pshvedko/nocopy/broker2/exchange"
	"github.com/pshvedko/nocopy/broker2/message"
)

type Transport map[string]uint

func (t Transport) Unsubscribe(u exchange.Topic) error {
	return u.Unsubscribe()
}

func (t Transport) Decode(ctx context.Context, bytes []byte, mediator message.Mediator) (context.Context, message.Message, error) {
	return message.Decode(ctx, bytes, mediator)
}

func (t Transport) Subscribe(ctx context.Context, at string, handler exchange.Handler) (exchange.Subscription, error) {
	t[at] = t[at] + 1
	return Subscription(at), nil
}

func (t Transport) QueueSubscribe(ctx context.Context, at string, queue string, handler exchange.Handler) (exchange.Subscription, error) {
	t[at] = t[at] + 1
	return Subscription(at), nil
}

func (t Transport) Prefix() [2]string {
	return [2]string{"#", "%"}
}

func (t Transport) Flush() error { return nil }

func (t Transport) Close() {}

type Subscription string

func (s Subscription) Drain() error { return nil }

func (s Subscription) Unsubscribe() error { return nil }

func TestExchange_Listen(t *testing.T) {
	transport := Transport{}
	e := exchange.New(transport)
	err := e.Listen(context.TODO(), "test", "host", "id")
	require.NoError(t, err)
	require.Equal(t,
		Transport{"#test": 1, "#test.host": 1, "#test.host.id": 1, "%test": 1, "%test.host": 1, "%test.host.id": 1},
		transport)
	clear(transport)
	err = e.Listen(context.TODO(), "test")
	require.NoError(t, err)
	require.Equal(t,
		Transport{"#test": 1, "%test": 1},
		transport)
}

func TestExchange_Shutdown(t *testing.T) {
	ctx := context.TODO()
	m := message.Envelope{
		ID:     uuid.New(),
		From:   "any",
		Return: []string{"path", "to", "home"},
		To:     "me",
		Type:   2,
		Method: "hello",
	}
	b, err := json.Marshal([]any{m, json.RawMessage{'{', '}'}})
	require.NoError(t, err)
	e := exchange.New(Transport{})
	e.Read(ctx, b)
	e.Shutdown()
}
