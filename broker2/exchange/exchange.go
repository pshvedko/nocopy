package exchange

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/pshvedko/nocopy/broker2/message"
)

type Subscription interface {
	Unsubscribe() error
	Drain() error
}

type Handler func(context.Context, string, []byte)

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
}

func New(transport Transport) *Exchange {
	return &Exchange{
		transport: transport,
	}
}

func (e *Exchange) Handle(by string, handler message.Handler) {
	//TODO implement me
	panic("implement me")
}

func (e *Exchange) Catch(by string, catcher message.Catcher) {
	//TODO implement me
	panic("implement me")
}

func (e *Exchange) Message(ctx context.Context, to string, by string, body message.Body, options ...any) (uuid.UUID, error) {
	//TODO implement me
	panic("implement me")
}

func (e *Exchange) Request(ctx context.Context, to string, by string, body message.Body, options ...any) (uuid.UUID, message.Message, error) {
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

func (e *Exchange) Middleware(topic string, handle string) ([]message.Middleware, error) {
	// FIXME
	return nil, nil
}

func (e *Exchange) Read(ctx context.Context, topic string, bytes []byte) {
	ctx, m, err := e.transport.Decode(ctx, topic, bytes, e)
	if err != nil {
		return
	}

	ctx, cancel := context.WithCancel(ctx)

	e.child.Add(1)
	e.child.Store(ctx, cancel)

	go func() {

		_ = m // _, _ = handler(ctx) // FIXME

		<-ctx.Done()

		<-time.After(5 * time.Second)

		e.child.Delete(ctx)
		e.child.Done()
		cancel()
	}()
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
