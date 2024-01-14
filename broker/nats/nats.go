package nats

import (
	"context"
	"errors"
	"log/slog"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/nats-io/nats.go"

	"github.com/pshvedko/nocopy/broker/message"
)

type Broker struct {
	*url.URL
	*nats.Conn
	handler      map[string]message.Handler
	catcher      map[string]message.Catcher
	subscription []*nats.Subscription
	topic        [3]string
	mu           sync.Mutex
}

func (b *Broker) Request(_ context.Context, to, by string, v any, oo ...any) (uuid.UUID, message.Message, error) {
	a, err := message.MakeAddress(b.At(3), oo)
	if err != nil {
		return uuid.UUID{}, nil, err
	}
	return b.request(message.Toward{To: to, By: by, Address: a}, Body{Any: v})
}

func (b *Broker) request(h message.Header, r message.Marshaler) (uuid.UUID, message.Message, error) {
	bytes, err := r.Marshal()
	if err != nil {
		return uuid.UUID{}, nil, err
	}
	of := h.OF()
	id := h.ID()
	slog.Warn("SYNC", "by", h.BY(), "id", id, "of", of, "at", h.AT(), "to", h.TO(), "path", h.RE())
	m, err := b.Conn.RequestMsg(&nats.Msg{
		Subject: h.TO(),
		Reply:   h.AT(),
		Header: nats.Header{
			"ID": strings.Split(id.String(), "-"),
			"BY": strings.Split(h.BY(), "."),
			"RE": h.RE(),
			"OF": of[:],
		},
		Data: bytes,
	}, time.Minute)
	if err != nil {
		return uuid.UUID{}, nil, err
	}
	return id, Message{m: m}, nil
}

func (b *Broker) Catch(method string, catcher message.Catcher) {
	b.catcher[method] = catcher
}

func (b *Broker) Handle(method string, handler message.Handler) {
	b.handler[method] = handler
}

func (b *Broker) Listen(_ context.Context, topic, host, id string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
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
	b.mu.Lock()
	defer b.mu.Unlock()
	if b == nil {
		return
	}
	for _, s := range b.subscription {
		_ = s.Unsubscribe()
		slog.Warn("FINISH", "at", s.Subject)
	}
	b.subscription = nil
}

func (b *Broker) Shutdown() {
	if b == nil {
		return
	}
	b.Finish()
	b.Conn.Close()
}

type Body struct {
	Any any
}

func (b Body) Marshal() ([]byte, error) {
	return message.Marshal(b.Any)
}

type Message struct {
	m *nats.Msg
}

func (m Message) Marshal() ([]byte, error) {
	return m.m.Data, nil
}

func (m Message) OF() [3]string {
	var ff [3]string
	copy(ff[:], m.m.Header["OF"])
	return ff
}

func (m Message) AT() string {
	return m.m.Subject
}

func (m Message) TO() string {
	return m.m.Reply
}

func (m Message) RE() []string {
	return m.m.Header["RE"]
}

func (m Message) BY() string {
	return strings.Join(m.m.Header["BY"], ".")
}

func (m Message) ID() uuid.UUID {
	id, _ := uuid.Parse(strings.Join(m.m.Header["ID"], "-"))
	return id
}

func (m Message) Unmarshal(a any) error {
	of := m.OF()
	if len(of[1]) > 0 {
		return errors.New(of[1])
	}
	return message.Unmarshal(m.m.Data, a)
}

func (b *Broker) onMessage(m *nats.Msg) {
	q := message.New().WithMessage(Message{m: m})
	by := q.BY()
	of := q.OF()
	slog.Warn("READ", "by", by, "id", q.ID(), "of", of, "at", q.AT(), "from", q.TO(), "path", q.RE())
	switch of[0] {
	case "F":
		h, ok := b.handler[by]
		if ok {
			r, err := h(context.TODO(), q)
			switch {
			case err != nil:
				q = q.WithError(err)
				fallthrough
			case r != nil:
				_, _ = b.send(message.Respond{Header: q}, Body{Any: r})
			}
		}
	case "R":
		c, ok := b.catcher[by]
		if ok {
			c(context.TODO(), q)
		}
	}
}

func (b *Broker) send(h message.Header, r message.Marshaler) (uuid.UUID, error) {
	bytes, err := r.Marshal()
	if err != nil {
		return uuid.UUID{}, err
	}
	of := h.OF()
	id := h.ID()
	slog.Warn("SEND", "by", h.BY(), "id", id, "of", of, "at", h.AT(), "to", h.TO(), "path", h.RE())
	return id, b.Conn.PublishMsg(&nats.Msg{
		Subject: h.TO(),
		Reply:   h.AT(),
		Header: nats.Header{
			"ID": strings.Split(id.String(), "-"),
			"BY": strings.Split(h.BY(), "."),
			"RE": h.RE(),
			"OF": of[:],
		},
		Data: bytes,
	})
}

func (b *Broker) Message(_ context.Context, to, by string, v any, oo ...any) (uuid.UUID, error) {
	a, err := message.MakeAddress(b.At(3), oo)
	if err != nil {
		return uuid.UUID{}, err
	}
	return b.send(message.Toward{To: to, By: by, Address: a}, Body{Any: v})
}

func (b *Broker) Send(_ context.Context, m message.Message, oo ...any) (uuid.UUID, error) {
	h, err := message.MakeHeader(m, oo...)
	if err != nil {
		return uuid.UUID{}, err
	}
	return b.send(h, m)
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
