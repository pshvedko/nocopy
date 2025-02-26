package exchange

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/google/uuid"

	"github.com/pshvedko/nocopy/broker/message"
	"github.com/pshvedko/nocopy/internal"
)

type Subscription interface {
	Unsubscribe() error
	Drain() error
}

type Transport interface {
	Flush() error
	Subscribe(context.Context, string, message.Decoder) (Subscription, error)
	QueueSubscribe(context.Context, string, string, message.Decoder) (Subscription, error)
	Publish(context.Context, message.Message, message.Encoder) error
	Unsubscribe(Topic) error
	Close()
}

type Topic struct {
	subject string
	wide    bool
	Subscription
}

func (t Topic) String() string {
	return t.subject
}

func (t Topic) Wide() bool {
	return t.wide
}

type Key struct {
	id     uuid.UUID
	method string
}

type Exchange struct {
	transport Transport
	topic     [2][]Topic
	child     internal.Map[context.Context, context.CancelFunc]
	reply     internal.Map[Key, chan<- message.Message]
	handler   map[string]message.HandleFunc
	catcher   map[string]message.CatchFunc
	functor   map[string][]message.Middleware
	wrapper   []message.Middleware
	options   []Option
	finish    atomic.Bool
	config    Config
}

func (e *Exchange) Encode(ctx context.Context, m message.Message) ([]byte, error) {
	return message.Encode(ctx, m, e)
}

func (e *Exchange) Decode(ctx context.Context, bytes []byte) (context.Context, message.Message, error) {
	return message.Decode(ctx, bytes, e)
}

func (e *Exchange) UseOptions(options ...Option) {
	e.options = append(e.options, options...)
}

func (e *Exchange) Topic(i int) (int, string) {
	n := len(e.topic[0]) - 1
	if i < 0 || i > n {
		i = n
	}
	return n, e.topic[0][i].subject
}

func (e *Exchange) UseTransport(transport Transport) {
	e.transport = transport
}

func (e *Exchange) Transport() Transport {
	return e.transport
}

func New(transport Transport) *Exchange {
	return &Exchange{
		transport: transport,
		handler:   map[string]message.HandleFunc{},
		catcher:   map[string]message.CatchFunc{},
		functor:   map[string][]message.Middleware{},
	}
}

func (e *Exchange) Middleware(method string) []message.Middleware {
	return append(e.wrapper, e.functor[method]...)
}

func (e *Exchange) UseMiddleware(wrapper ...message.Middleware) {
	e.wrapper = append(e.wrapper, wrapper...)
}

func (e *Exchange) Handle(method string, handler message.HandleFunc) {
	e.handler[method] = handler
}

func (e *Exchange) Catch(method string, catcher message.CatchFunc) {
	e.catcher[method] = catcher
}

func (e *Exchange) Message(ctx context.Context, to string, method string, body message.Body, options ...Option) (uuid.UUID, error) {
	m := message.New().
		WithMethod(method).
		WithBody(body).
		WithFrom(e.topic[0][0].subject).
		WithTo(to).
		Build()

	return e.Send(ctx, m, options...)
}

func (e *Exchange) Request(ctx context.Context, to string, method string, body message.Body, options ...Option) (message.Message, error) {
	c := make(chan message.Message, 1)
	m := message.New().
		WithType(message.Request).
		WithMethod(method).
		WithBody(body).
		WithFrom(e.topic[0][0].subject).
		WithTo(to).
		Build()
	o, m := e.Apply(m, options...)
	k := Key{
		id:     m.ID(),
		method: method,
	}

	if !e.reply.Store(k, c) {
		return nil, message.ErrIllegalID
	}

	if o.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, o.Timeout)
		defer cancel()
	}

	_, err := e.Send(ctx, m, options...)
	if err == nil {
		select {
		case m = <-c:
			return m, nil
		case <-ctx.Done():
			err = ctx.Err()
		}
	}

	e.reply.Delete(k)

	return nil, err
}

func (e *Exchange) Answer(ctx context.Context, m message.Message, body message.Body, options ...Option) (uuid.UUID, error) {
	if m.Type()&message.Answer == message.Answer {
		return uuid.UUID{}, message.ErrIllegalType
	}

	m = message.NewMessage(m).
		WithBody(body).
		Answer()

	return e.Send(ctx, m, options...)
}

func (e *Exchange) Forward(ctx context.Context, to string, m message.Message, options ...Option) (uuid.UUID, error) {
	if m.Type()&message.Answer == message.Answer {
		return uuid.UUID{}, message.ErrIllegalType
	}

	m = message.NewMessage(m).
		Forward(to)

	return e.Send(ctx, m, options...)
}

func (e *Exchange) Backward(ctx context.Context, m message.Message, options ...Option) (uuid.UUID, error) {
	if m.Type()&message.Answer == message.Query {
		return uuid.UUID{}, message.ErrIllegalType
	}
	if len(m.Return()) == 0 {
		return uuid.UUID{}, nil
	}

	m = message.NewMessage(m).
		Backward()

	return e.Send(ctx, m, options...)
}

func (e *Exchange) Send(ctx context.Context, m message.Message, options ...Option) (uuid.UUID, error) {
	if m.Type()&message.Answer == message.Answer && ctx.Value(message.Broadcast) != nil {
		return uuid.UUID{}, nil
	}

	_, m = e.Apply(m, e.options...)
	_, m = e.Apply(m, options...)

	return m.ID(), e.transport.Publish(ctx, m, e)
}

func (e *Exchange) Listen(ctx context.Context, on string, to ...string) error {
	at := on
	for {
		s, err := e.transport.QueueSubscribe(ctx, at, on, e)
		if err != nil {
			return err
		}

		e.topic[0] = append(e.topic[0], Topic{
			subject:      at,
			Subscription: s,
		})

		if len(to) == 0 {
			break
		}

		s, err = e.transport.Subscribe(ctx, at, e)
		if err != nil {
			return err
		}

		e.topic[1] = append(e.topic[1], Topic{
			subject:      at,
			wide:         true,
			Subscription: s,
		})

		at = fmt.Sprint(at, ".", to[0])
		to = to[1:]
	}

	return e.transport.Flush()
}

func (e *Exchange) Do(ctx context.Context, m message.Message) {
	ctx, cancel := context.WithCancel(context.WithValue(ctx, m.Type(), m.ID()))

	e.child.Store(ctx, cancel)

	go e.Run(ctx, m)
}

func (e *Exchange) Run(ctx context.Context, m message.Message) {
	b := message.NewMessage(m)

	switch m.Type() {
	case message.Query, message.Request, message.Broadcast:
		h, ok := e.handler[m.Method()]
		if ok {
			r, err := h(ctx, b)
			switch {
			case err != nil:
				_, _ = e.Send(ctx, b.WithError(err).Answer())
			case r != nil:
				_, _ = e.Send(ctx, b.WithBody(r).Answer())
			}
		}
	case message.Failure, message.Answer:
		q, ok := e.reply.Delete(Key{
			id:     m.ID(),
			method: m.Method(),
		})
		if ok {
			q <- m
			break
		}
		c, ok := e.catcher[m.Method()]
		if ok {
			if c(ctx, m) {
				break
			}
		}
		if len(m.Return()) > 0 {
			_, _ = e.Send(ctx, b.Backward())
		}
	}

	cancel, ok := e.child.Delete(ctx)
	if ok {
		cancel()
	}
}

func (e *Exchange) Finish() {
	defer e.child.Wait()
	defer e.reply.Wait()
	if !e.finish.CompareAndSwap(false, true) {
		return
	}
	for i, topic := range &e.topic {
		for _, t := range topic {
			_ = e.transport.Unsubscribe(t)
		}
		e.topic[i] = e.topic[i][:0]
	}
	e.child.Range(func(ctx context.Context, cancel context.CancelFunc) bool {
		cancel()
		return true
	})
}

func (e *Exchange) Shutdown() {
	e.Finish()
	e.transport.Close()
}
