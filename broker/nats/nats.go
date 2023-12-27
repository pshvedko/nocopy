package nats

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
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
	handler      map[string]message.Handler
	catcher      map[string]message.Catcher
	subscription []Subscriber
	topic        [3]string
}

func (b *Broker) Catch(method string, catcher message.Catcher) {
	b.catcher[method] = catcher
}

func (b *Broker) Handle(method string, handler message.Handler) {
	b.handler[method] = handler
}

func (b *Broker) Listen(_ context.Context, topic, host, id string) error {
	b.topic = [3]string{topic, host, id}
	for i := 1; i < 4; i++ {
		at := b.At(i)
		s, err := b.Conn.QueueSubscribe(at, topic, b.onMessage)
		if err != nil {
			return err
		}
		b.subscription = append(b.subscription, s)
		slog.Warn("LISTEN", "by", topic, "at", at)
	}
	return b.Conn.Flush()
}

func (b *Broker) Finish() {
	slog.Warn("FINISH")
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

type Query struct {
	m *nats.Msg
}

func (q Query) FF() [3]string {
	var ff [3]string
	copy(ff[:], q.m.Header["FF"])
	return ff
}

func (q Query) AT() string {
	return q.m.Subject
}

func (q Query) TO() string {
	return q.m.Reply
}

func (q Query) RE() []string {
	return q.m.Header["RE"]
}

func (q Query) BY() string {
	return strings.Join(q.m.Header["BY"], ".")
}

func (q Query) ID() uuid.UUID {
	id, _ := uuid.Parse(strings.Join(q.m.Header["ID"], "-"))
	return id
}

func (q Query) Unmarshal(a any) error {
	ff := q.FF()
	if len(ff[1]) > 0 {
		return errors.New(ff[1])
	}
	return json.Unmarshal(q.m.Data, a)
}

func (q Query) WithError(err error) Query {
	ff := q.FF()
	ff[1] = err.Error()
	q.m.Header["FF"] = ff[:]
	return q
}

func (b *Broker) onMessage(m *nats.Msg) {
	q := Query{m: m}
	by := q.BY()
	ff := q.FF()
	slog.Warn("READ", "by", by, "id", q.ID(), "flag", ff, "at", q.AT(), "to", q.TO(), "path", q.RE())
	switch ff[0] {
	case "F":
		h, ok := b.handler[by]
		if ok {
			r, err := h(context.TODO(), q)
			switch {
			case err != nil:
				q = q.WithError(err)
				fallthrough
			case r != nil:
				_, _ = b.send(message.Backward{Header: q}, message.Body{Any: r})
			}
		}
	case "R":
		c, ok := b.catcher[by]
		if ok {
			c(context.TODO(), q)
		}
	}
}

func (b *Broker) send(h message.Header, r message.Reply) (uuid.UUID, error) {
	bytes, err := r.Marshal()
	if err != nil {
		return uuid.UUID{}, err
	}
	ff := h.FF()
	id := h.ID()
	slog.Warn("SEND", "by", h.BY(), "id", id, "flag", ff, "at", h.AT(), "to", h.TO(), "path", h.RE())
	return id, b.Conn.PublishMsg(&nats.Msg{
		Subject: h.TO(),
		Reply:   h.AT(),
		Header: nats.Header{
			"ID": strings.Split(id.String(), "-"),
			"BY": strings.Split(h.BY(), "."),
			"RE": h.RE(),
			"FF": ff[:],
		},
		Data: bytes,
	})
}

func (b *Broker) Send(_ context.Context, to, by string, v any, oo ...any) (id uuid.UUID, err error) {
	a, err := message.MakeAddress(b.At(3), oo)
	if err != nil {
		return
	}
	return b.send(message.Toward{To: to, By: by, Address: a}, message.Body{Any: v})
}

func (b *Broker) At(n int) string {
	return strings.Join(b.topic[:n], ":")
}

func New(ur1 *url.URL) (*Broker, error) {
	conn, err := nats.Connect(ur1.String(), nats.NoEcho())
	if err != nil {
		return nil, err
	}
	return &Broker{
		URL:     ur1,
		Conn:    conn,
		handler: make(map[string]message.Handler),
		catcher: make(map[string]message.Catcher),
	}, nil
}
