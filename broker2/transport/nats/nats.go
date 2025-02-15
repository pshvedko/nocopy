package nats

import (
	"context"
	"net/url"

	"github.com/nats-io/nats.go"

	"github.com/pshvedko/nocopy/broker2/exchange"
	"github.com/pshvedko/nocopy/broker2/message"
)

type Transport struct {
	conn *nats.Conn
	message.Formatter
}

func (t *Transport) Unsubscribe(topic exchange.Topic) error {
	return topic.Unsubscribe()
}

func (t *Transport) Publish(ctx context.Context, m message.Message, w message.Mediator) error {
	to, bytes, err := t.Encode(ctx, m, w)
	if err != nil {
		return err
	}
	return t.conn.Publish(to, bytes)
}

func (t *Transport) Subscribe(ctx context.Context, at string, w message.Mediator, r exchange.Doer) (exchange.Subscription, error) {
	return t.conn.Subscribe(at, func(m *nats.Msg) {
		ctx2, z, err := t.Decode(ctx, m.Data, w)
		if err != nil {
			return
		}
		r.Do(ctx2, z)
	})
}

func (t *Transport) QueueSubscribe(ctx context.Context, at string, queue string, w message.Mediator, r exchange.Doer) (exchange.Subscription, error) {
	return t.conn.QueueSubscribe(at, queue, func(m *nats.Msg) {
		ctx2, q, err := t.Decode(ctx, m.Data, w)
		if err != nil {
			return
		}
		r.Do(ctx2, q)
	})
}

func (t *Transport) Flush() error {
	return t.conn.Flush()
}

func (t *Transport) Close() {
	t.conn.Close()
}

func New(u *url.URL, decoder message.Formatter) (*Transport, error) {
	c, err := nats.Connect(u.String(), nats.NoEcho())
	if err != nil {
		return nil, err
	}
	return &Transport{
		conn:      c,
		Formatter: decoder,
	}, nil
}
