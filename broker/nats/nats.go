package nats

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"net/url"

	"github.com/pshvedko/nocopy/broker/message"
)

type Subscriber interface {
	Unsubscribe() error
}

type Broker struct {
	*url.URL
	*nats.Conn
	handler      map[string]func(context.Context, message.Query) (message.Reply, error)
	subscription []Subscriber
	topic        string
}

func (b *Broker) Handle(method string, handler func(context.Context, message.Query) (message.Reply, error)) {
	b.handler[method] = handler
}

func (b *Broker) Listen(ctx context.Context, topic string, wait bool) error {
	b.Topic(topic)
	s, err := b.Conn.QueueSubscribe(topic, topic, b.Message)
	if err != nil {
		return err
	}
	b.subscription = append(b.subscription, s)
	if !wait {
		return nil
	}
	<-ctx.Done()
	b.subscription = b.subscription[:0]
	return s.Unsubscribe()
}

type Query struct {
	m *nats.Msg
}

func (q Query) Unmarshal(a any) error {
	return json.Unmarshal(q.m.Data, a)
}

func (b *Broker) Message(m *nats.Msg) {
	_ = m.InProgress()
	method := m.Header.Get("BY")
	handle, ok := b.handler[method]
	if ok {
		_, _ = handle(context.Background(), Query{m: m})
	}
	_ = m.Ack()
}

func (b *Broker) Send(_ context.Context, to, by string, a any) (id uuid.UUID, err error) {
	bytes, err := json.Marshal(a)
	if err != nil {
		return
	}
	id = uuid.New()
	err = b.Conn.PublishMsg(&nats.Msg{
		Subject: to,
		Reply:   b.topic,
		Header:  nats.Header{"BY": []string{by}, "ID": []string{id.String()}},
		Data:    bytes,
	})
	return
}

func (b *Broker) Shutdown(_ context.Context) (err error) {
	i := len(b.subscription)
	for i > 0 {
		i--
		err = b.subscription[i].Unsubscribe()
		if err != nil {
			return
		}
		b.subscription = b.subscription[:i]
	}
	b.Close()
	return
}

func (b *Broker) Topic(topic string) {
	b.topic = topic
}

func New(ur1 *url.URL) (*Broker, error) {
	conn, err := nats.Connect(ur1.String())
	if err != nil {
		return nil, err
	}
	return &Broker{
		URL:     ur1,
		Conn:    conn,
		handler: make(map[string]func(context.Context, message.Query) (message.Reply, error)),
	}, nil
}
