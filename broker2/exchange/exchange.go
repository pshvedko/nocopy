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

type Handler func(context.Context, []byte)

type Transport interface {
	message.Decoder
	Flush() error
	Subscribe(context.Context, string, Handler) (Subscription, error)
	QueueSubscribe(context.Context, string, string, Handler) (Subscription, error)
	Unsubscribe(Topic) error
	Prefix() [2]string
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
	functor   map[string][]message.MiddlewareFunc
	wrapper   []message.MiddlewareFunc
}

func New(transport Transport) *Exchange {
	return &Exchange{
		transport: transport,
		topic:     [2][]Topic{},
		child:     Child{},
		handler:   map[string]message.HandleFunc{},
		catcher:   map[string]message.CatchFunc{},
		functor:   map[string][]message.MiddlewareFunc{},
		wrapper:   []message.MiddlewareFunc{},
	}
}

func (e *Exchange) Use(wrapper message.MiddlewareFunc) {
	e.wrapper = append(e.wrapper, wrapper)
}

func (e *Exchange) Middleware(method string) []message.MiddlewareFunc {
	return append(e.wrapper, e.functor[method]...)
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

func (e *Exchange) Send(ctx context.Context, message message.Message, options ...any) (uuid.UUID, error) {
	//TODO implement me
	panic("implement me")
}

// Listen ...
func (e *Exchange) Listen(ctx context.Context, at string, to ...string) error {
	p := e.transport.Prefix()
	a := fmt.Sprint(p[0], at)
	u := fmt.Sprint(p[1], at)

	for {
		s, err := e.transport.Subscribe(ctx, a, e.Read)
		if err != nil {
			return err
		}

		e.topic[0] = append(e.topic[0], Topic{
			subject:      a,
			Subscription: s,
		})

		s, err = e.transport.QueueSubscribe(ctx, u, at, e.Read)
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

func (e *Exchange) Read(ctx context.Context, bytes []byte) {
	ctx, m, err := e.transport.Decode(ctx, bytes, e)
	if err != nil {
		return
	}

	ctx, cancel := context.WithCancel(ctx)

	e.child.Add(1)
	e.child.Store(ctx, cancel)

	go e.Do(ctx, cancel, message.New(m))
}

func (e *Exchange) Do(ctx context.Context, cancel context.CancelFunc, m message.Builder) {
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
				_, _ = e.Send(ctx, m.WithBody(r)) // FIXME
			}
		}
	case message.Failure, message.Reply:
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
