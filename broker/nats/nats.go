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
	topic        [3]string
}

func (b *Broker) Handle(method string, handler func(context.Context, message.Query) (message.Reply, error)) {
	b.handler[method] = handler
}

func (b *Broker) Listen(_ context.Context, topic, host, id string) error {
	b.topic = [3]string{topic, host, id}
	for i := 1; i < 4; i++ {
		at := b.At(i)
		println("=============== LISTEN", at, topic)
		s, err := b.Conn.QueueSubscribe(at, topic, b.onMessage)
		if err != nil {
			return err
		}
		b.subscription = append(b.subscription, s)
	}
	return b.Conn.Flush()
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

func (b *Broker) onMessage(m *nats.Msg) {
	_ = m.InProgress()
	q := Query{m: m}
	h, ok := b.handler[q.BY()]
	if ok {
		_, _ = h(context.TODO(), q)
	}
	_ = m.Ack()
}

func (b *Broker) Query(_ context.Context, to, by string, a any, oo ...any) (id uuid.UUID, err error) {
	o, err := message.Options(b.At(3), oo)
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

func (b *Broker) Finish() {
	println("=============== FINISH")
	if b == nil {
		return
	}
	for i := range b.subscription {
		_ = b.subscription[i].Unsubscribe()
	}
	b.subscription = b.subscription[:0]
}

func (b *Broker) Shutdown() {
	if b == nil {
		return
	}
	b.Finish()
	b.Conn.Close()
}

func (b *Broker) At(n int) string {
	return strings.Join(b.topic[:n], ":")
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
