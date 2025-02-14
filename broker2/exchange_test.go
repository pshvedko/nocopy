package exchange_test

import (
	"context"
	"github.com/pshvedko/nocopy/broker2"
	"github.com/pshvedko/nocopy/broker2/exchange/message"
	"testing"

	"github.com/stretchr/testify/require"
)

type Transport map[string]uint

func (t Transport) Flush() error { return nil }

type Subscription string

func (s Subscription) Unsubscribe() error { return nil }

func (t Transport) Subscribe(at string, f func(m message.Message)) (exchange.Subscription, error) {
	t[at] = t[at] + 1
	return Subscription(at), nil
}

func (t Transport) QueueSubscribe(at string, queue string, f func(m message.Message)) (exchange.Subscription, error) {
	t[at] = t[at] + 1
	return Subscription(at), nil
}

func TestExchange_Listen(t *testing.T) {
	transport := Transport{}
	b := exchange.NewExchange(transport)
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
