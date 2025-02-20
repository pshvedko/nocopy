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
}

func (t *Transport) Unsubscribe(topic exchange.Topic) error {
	return topic.Unsubscribe()
}

const TopicPrefix = "%@"
const TopicLength = 1

func (t *Transport) Publish(ctx context.Context, m message.Message, e message.Encoder) error {
	bytes, err := e.Encode(ctx, m)
	if err != nil {
		return err
	}
	if m.Type() == message.Broadcast {
		return t.conn.Publish(TopicPrefix[:TopicLength]+m.To(), bytes)
	}
	return t.conn.Publish(TopicPrefix[TopicLength:]+m.To(), bytes)
}

func (t *Transport) Subscribe(ctx context.Context, at string, d message.Decoder) (exchange.Subscription, error) {
	return t.conn.Subscribe(TopicPrefix[:TopicLength]+at, func(m *nats.Msg) {
		ctx2, z, err := d.Decode(ctx, m.Data)
		if err != nil {
			return
		}
		d.Do(ctx2, z)
	})
}

func (t *Transport) QueueSubscribe(ctx context.Context, at string, queue string, d message.Decoder) (exchange.Subscription, error) {
	return t.conn.QueueSubscribe(TopicPrefix[TopicLength:]+at, queue, func(m *nats.Msg) {
		ctx2, z, err := d.Decode(ctx, m.Data)
		if err != nil {
			return
		}
		d.Do(ctx2, z)
	})
}

func (t *Transport) Flush() error {
	return t.conn.Flush()
}

func (t *Transport) Close() {
	t.conn.Close()
}

func New(u *url.URL) (*Transport, error) {
	c, err := nats.Connect(u.String(), nats.NoEcho())
	if err != nil {
		return nil, err
	}
	return &Transport{
		conn: c,
	}, nil
}
