package exchange

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"sync"

	"github.com/pshvedko/nocopy/broker2/message"
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
	Subscription
}

func (t Topic) String() string {
	return t.subject
}

type Map[K comparable, V any] struct {
	m sync.Map
}

func (s *Map[K, V]) Store(key K, value V) {
	s.m.Store(key, value)
}

func (s *Map[K, V]) Delete(key K) bool {
	_, ok := s.m.LoadAndDelete(key)
	return ok
}

func (s *Map[K, V]) Range(f func(key K, value V) bool) {
	s.m.Range(func(key, value any) bool {
		return f(key.(K), value.(V))
	})
}

type Child struct {
	sync.WaitGroup
	Map[context.Context, context.CancelFunc]
}

type Exchange struct {
	transport Transport
	topic     [2][]Topic
	child     Child
	handler   map[string]message.HandleFunc
	catcher   map[string]message.CatchFunc
	functor   map[string][]message.Middleware
	wrapper   []message.Middleware
}

func New(transport Transport) *Exchange {
	return &Exchange{
		transport: transport,
		topic:     [2][]Topic{},
		child:     Child{},
		handler:   map[string]message.HandleFunc{},
		catcher:   map[string]message.CatchFunc{},
		functor:   map[string][]message.Middleware{},
		wrapper:   []message.Middleware{},
	}
}

func (e *Exchange) Middleware(method string) []message.Middleware {
	return append(e.wrapper, e.functor[method]...)
}

func (e *Exchange) Use(wrapper message.Middleware) {
	e.wrapper = append(e.wrapper, wrapper)
}

func (e *Exchange) Handle(method string, handler message.HandleFunc) {
	e.handler[method] = handler
}

func (e *Exchange) Catch(method string, catcher message.CatchFunc) {
	e.catcher[method] = catcher
}

func (e *Exchange) Message(ctx context.Context, to string, method string, body message.Body, options ...any) (uuid.UUID, error) {
	//TODO implement me
	panic("implement me")
}

func (e *Exchange) Request(ctx context.Context, to string, method string, body message.Body, options ...any) (uuid.UUID, message.Message, error) {
	//TODO implement me
	panic("implement me")
}

func (e *Exchange) Send(ctx context.Context, m message.Message, options ...any) (id uuid.UUID, err error) {
	id = m.ID()
	err = e.transport.Publish(ctx, m, e)
	return
}

func (e *Exchange) Listen(ctx context.Context, at string, to ...string) error {
	a := at
	u := at

	for {
		s, err := e.transport.Subscribe(ctx, a, e, e)
		if err != nil {
			return err
		}

		e.topic[0] = append(e.topic[0], Topic{
			subject:      a,
			Subscription: s,
		})

		s, err = e.transport.QueueSubscribe(ctx, u, at, e, e)
		if err != nil {
			return err
		}

		e.topic[1] = append(e.topic[1], Topic{
			subject:      u,
			Subscription: s,
		})

		if len(to) == 0 {
			break
		}

		a = fmt.Sprint(a, ".", to[0])
		u = fmt.Sprint(u, ".", to[0])

		to = to[1:]
	}

	return e.transport.Flush()
}

func (e *Exchange) Do(ctx context.Context, m message.Message) {
	ctx, cancel := context.WithCancel(ctx)

	e.child.Add(1)
	e.child.Store(ctx, cancel)

	go e.Run(ctx, cancel, message.New(m))
}

func (e *Exchange) Run(ctx context.Context, cancel context.CancelFunc, m message.Builder) {
	switch m.Type() {
	case message.Query, message.Broadcast:
		h, ok := e.handler[m.Method()]
		if ok {
			r, err := h(ctx, m)
			switch {
			case err != nil:
				m = m.WithError(err)
				fallthrough
			case r != nil:
				if m.Type() == message.Query {
					_, _ = e.Send(ctx, m.Answer().WithBody(r)) // FIXME
				}
			}
		}
	case message.Failure, message.Answer:
		c, ok := e.catcher[m.Method()]
		if ok {
			c(ctx, m)
		}
	}
	e.child.Delete(ctx)
	e.child.Done()
	cancel()
}

func (e *Exchange) Finish() {
	for _, topic := range &e.topic {
		for _, t := range topic {
			_ = e.transport.Unsubscribe(t)
		}
	}
	e.child.Range(func(ctx context.Context, cancel context.CancelFunc) bool {
		ok := e.child.Delete(ctx)
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
