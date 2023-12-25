package nats

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"

	"github.com/google/uuid"

	"github.com/nats-io/nats.go"

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

func (q Query) AT() string {
	return q.m.Subject
}

func (q Query) RE() []string {
	return append(q.m.Header["RE"], q.m.Reply)
}

func (q Query) BY() string {
	return strings.Join(q.m.Header["BY"], "!")
}

func (q Query) ID() uuid.UUID {
	var id uuid.UUID
	copy(id[:], q.m.Header.Get("ID"))
	return id
}

func (q Query) Unmarshal(a any) error {
	return json.Unmarshal(q.m.Data, a)
}

func (b *Broker) Message(m *nats.Msg) {
	_ = m.InProgress()
	q := Query{m: m}
	h, ok := b.handler[q.BY()]
	if ok {
		_, _ = h(context.TODO(), q)
	}
	_ = m.Ack()
}

func (b *Broker) Query(_ context.Context, to, by string, a any, oo ...any) (id uuid.UUID, err error) {
	o, err := message.Options(b.topic, oo)
	if err != nil {
		return
	}
	bytes, err := json.Marshal(a)
	if err != nil {
		return
	}
	id = o.ID()
	err = b.Conn.PublishMsg(&nats.Msg{
		Subject: to,
		Reply:   o.AT(),
		Header: nats.Header{
			"BY": []string{by},
			"ID": []string{string(id[:])},
			"RE": o.RE(),
		},
		Data: bytes,
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
