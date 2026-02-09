package nats

import (
	"context"
	"net/url"

	"github.com/nats-io/nats.go"

	"github.com/pshvedko/nocopy/broker/exchange"
	"github.com/pshvedko/nocopy/broker/message"
)

type Transport struct {
	conn *nats.Conn
}

func (t *Transport) Unsubscribe(topic exchange.Topic) error {
	return topic.Unsubscribe()
}

const TopicPrefix = "%@"

func (t *Transport) Publish(ctx context.Context, m message.Message, e message.Encoder) error {
	headers, bytes, err := e.Encode(ctx, m)
	if err != nil {
		return err
	}
	if m.Type() == message.Broadcast {
		return t.conn.PublishMsg(Message(TopicPrefix[:1]+m.To(), headers, bytes))
	}
	return t.conn.PublishMsg(Message(TopicPrefix[1:]+m.To(), headers, bytes))
}

func (t *Transport) Subscribe(ctx context.Context, at string, d message.Decoder) (exchange.Subscription, error) {
	return t.conn.Subscribe(TopicPrefix[:1]+at, func(m *nats.Msg) {
		ctx2, z, err := d.Decode(ctx, m.Header, m.Data)
		if err != nil {
			return
		}
		d.Do(ctx2, z)
	})
}

func (t *Transport) QueueSubscribe(ctx context.Context, at string, queue string, d message.Decoder) (exchange.Subscription, error) {
	return t.conn.QueueSubscribe(TopicPrefix[1:]+at, queue, func(m *nats.Msg) {
		ctx2, z, err := d.Decode(ctx, m.Header, m.Data)
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

func New(u *url.URL, name string) (*Transport, error) {
	c, err := nats.Connect(u.String(), nats.Name(name), nats.NoEcho())
	if err != nil {
		return nil, err
	}
	return &Transport{
		conn: c,
	}, nil
}

func Message(subject string, headers map[string][]string, bytes []byte) *nats.Msg {
	return &nats.Msg{Subject: subject, Header: headers, Data: bytes}
}
