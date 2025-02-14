package nats

import (
	"net/url"

	"github.com/nats-io/nats.go"

	"github.com/pshvedko/nocopy/broker2/exchange"
	"github.com/pshvedko/nocopy/broker2/message"
)

type Transport struct {
	conn *nats.Conn
}

func (t *Transport) Flush() error {
	return t.conn.Flush()
}

type Message struct {
	Data []byte
}

func (t *Transport) Subscribe(at string, f func(string, message.Message)) (exchange.Subscription, error) {
	return t.conn.Subscribe(at, func(m *nats.Msg) {
		go f(m.Subject, Message{Data: m.Data})
	})
}

func (t *Transport) QueueSubscribe(at string, queue string, f func(string, message.Message)) (exchange.Subscription, error) {
	return t.conn.QueueSubscribe(at, queue, func(m *nats.Msg) {
		go f(m.Subject, Message{Data: m.Data})
	})
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
