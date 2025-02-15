package exchange_test

import (
	"context"
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

func (t Transport) Subscribe(_ context.Context, at string, _ message.Mediator, _ exchange.Doer) (exchange.Subscription, error) {
	t[at] = t[at] + 1
	return Subscription(at), nil
}

func (t Transport) QueueSubscribe(_ context.Context, at string, _ string, _ message.Mediator, _ exchange.Doer) (exchange.Subscription, error) {
	t[at] = t[at] + 1
	return Subscription(at), nil
}

func (t Transport) Flush() error { return nil }

func (t Transport) Close() {}

type Subscription string

func (s Subscription) Drain() error { return nil }

func (s Subscription) Unsubscribe() error { return nil }

func (t Transport) Publish(context.Context, message.Message, message.Mediator) error {
	//TODO implement me
	panic("implement me")
}

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
	transport := Transport{}
	e := exchange.New(transport)
	e.Do(context.TODO(), message.Raw{
		Envelope: message.Envelope{
			ID:     uuid.New(),
			From:   "any",
			Return: []string{"path", "to", "home"},
			To:     "me",
			Type:   2,
			Method: "hello",
		},
	})
	e.Shutdown()
}
