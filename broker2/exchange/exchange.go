package exchange

import (
	"context"
	"fmt"
	"github.com/google/uuid"

	"github.com/pshvedko/nocopy/broker2/message"
	"github.com/pshvedko/nocopy/internal"
)

type Subscription interface {
	Unsubscribe() error
	Drain() error
}

type Doer interface {
	Do(context.Context, message.Message)
}

type Transport interface {
	Flush() error
	Subscribe(context.Context, string, message.Mediator, Doer) (Subscription, error)
	QueueSubscribe(context.Context, string, string, message.Mediator, Doer) (Subscription, error)
	Publish(context.Context, message.Message, message.Mediator) error
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
	//TODO implement me
	panic("implement me")
}

func (e *Exchange) Request(ctx context.Context, to string, method string, body message.Body, options ...Option) (message.Message, error) {
	c := make(chan message.Message, 1)
	m := message.New().
		WithMethod(method).
		WithBody(body).
		WithTo(to).
		Build()
	m = e.Apply(m, options...)
	k := Key{
		id:     m.ID(),
		method: method,
	}

	if e.reply.Store(k, c) {
		return nil, message.ErrIllegalID
	}

	_, err := e.Send(ctx, m)
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
	b := message.NewMessage(m).Answer().WithBody(body)
	return e.Send(ctx, b.Build(), options...)
}

func (e *Exchange) Send(ctx context.Context, m message.Message, options ...Option) (uuid.UUID, error) {
	if ctx.Value(message.Broadcast) != nil && m.Type()&message.Answer == message.Answer {
		return uuid.UUID{}, nil
	}
	m = e.Apply(m, e.options...)
	m = e.Apply(m, options...)
	return m.ID(), e.transport.Publish(ctx, m, e)
}

func (e *Exchange) Listen(ctx context.Context, on string, to ...string) error {
	at := on
	for {
		s, err := e.transport.QueueSubscribe(ctx, at, on, e, e)
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

		s, err = e.transport.Subscribe(ctx, at, e, e)
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

	go e.Run(ctx, cancel, m)
}

func (e *Exchange) Run(ctx context.Context, cancel context.CancelFunc, m message.Message) {
	switch m.Type() {
	case message.Query, message.Synchro, message.Broadcast:
		h, ok := e.handler[m.Method()]
		if ok {
			b := message.NewMessage(m)
			r, err := h(ctx, b)
			switch {
			case err != nil:
				b = b.WithError(err)
				fallthrough
			case r != nil:
				_, _ = e.Answer(ctx, b.Build(), r)
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
			c(ctx, m)
		}
	}
	e.child.Delete(ctx)
	cancel()
}

func (e *Exchange) Finish() {
	for _, topic := range &e.topic {
		for _, t := range topic {
			_ = e.transport.Unsubscribe(t)
		}
	}
	e.child.Range(func(ctx context.Context, cancel context.CancelFunc) bool {
		_, ok := e.child.Delete(ctx)
		if ok {
			cancel()
		}
		return true
	})
	e.child.Wait()
	e.topic[0] = e.topic[0][:0]
	e.topic[1] = e.topic[1][:0]
}

func (e *Exchange) Shutdown() {
	e.Finish()
	e.transport.Close()
}
