package exchange_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pshvedko/nocopy/broker2/exchange"
	"github.com/pshvedko/nocopy/broker2/message"
)

type Transport map[string]uint

func (t Transport) Flush() error { return nil }

type Subscription string

func (s Subscription) Unsubscribe() error { return nil }

func (t Transport) Subscribe(at string, f func(string, message.Message)) (exchange.Subscription, error) {
	t[at] = t[at] + 1
	return Subscription(at), nil
}

func (t Transport) QueueSubscribe(at string, queue string, f func(string, message.Message)) (exchange.Subscription, error) {
	t[at] = t[at] + 1
	return Subscription(at), nil
}

func TestExchange_Listen(t *testing.T) {
	transport := Transport{}
	b := exchange.New(transport)
	err := b.Listen(context.TODO(), "test", "host", "id")
	require.NoError(t, err)
	require.Equal(t,
		Transport{"#test": 1, "#test.host": 1, "#test.host.id": 1, "@test": 1, "@test.host": 1, "@test.host.id": 1},
		transport)
	clear(transport)
	err = b.Listen(context.TODO(), "test")
	require.NoError(t, err)
	require.Equal(t,
		Transport{"#test": 1, "@test": 1},
		transport)
}
