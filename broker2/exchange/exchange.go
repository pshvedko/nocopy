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
	wide    bool
	Subscription
}

func (t Topic) String() string {
	return t.subject
}

func (t Topic) Wide() bool {
	return t.wide
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

type Config struct {
	MaxRequestTopic bool
	MaxRespondTopic bool
}

type Exchange struct {
	transport Transport
	config    Config
	topic     [2][]Topic
	child     Child
	handler   map[string]message.HandleFunc
	catcher   map[string]message.CatchFunc
	functor   map[string][]message.Middleware
	wrapper   []message.Middleware
}

func (e *Exchange) MaxRequestTopic(enabled bool) {
	e.config.MaxRequestTopic = enabled
}

func (e *Exchange) MaxRespondTopic(enabled bool) {
	e.config.MaxRespondTopic = enabled
}

func (e *Exchange) Topic(i int) (int, string) {
	n := len(e.topic[0]) - 1
	if i < 0 || i > n {
		i = n
	}
	return n, e.topic[0][i].subject
}

func (e *Exchange) Wrap(transport Transport) {
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

func (e *Exchange) Use(wrapper message.Middleware) {
	e.wrapper = append(e.wrapper, wrapper)
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

func (e *Exchange) Request(ctx context.Context, to string, method string, body message.Body, options ...Option) (uuid.UUID, message.Message, error) {
	//TODO implement me
	panic("implement me")
}

func (e *Exchange) Answer(ctx context.Context, m message.Message, body message.Body, options ...Option) (uuid.UUID, error) {
	b := message.NewMessage(m).Answer().WithBody(body)
	if e.config.MaxRespondTopic {
		options = append(options, WithMaxFrom())
	}
	return e.Send(ctx, b.Build(), options...)
}

func (e *Exchange) Send(ctx context.Context, m message.Message, options ...Option) (uuid.UUID, error) {
	if ctx.Value(message.Broadcast) != nil && m.Type()&message.Answer == message.Answer {
		return uuid.UUID{}, nil
	}
	return m.ID(), e.transport.Publish(ctx, e.Apply(m, options...), e)
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

	e.child.Add(1)
	e.child.Store(ctx, cancel)

	go e.Run(ctx, cancel, m)
}

func (e *Exchange) Run(ctx context.Context, cancel context.CancelFunc, m message.Message) {
	b := message.NewMessage(m)
	switch b.Type() {
	case message.Query, message.Synchro, message.Broadcast:
		h, ok := e.handler[b.Method()]
		if ok {
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
		c, ok := e.catcher[b.Method()]
		if ok {
			c(ctx, b.Build())
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
