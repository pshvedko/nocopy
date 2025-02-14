package exchange

import (
	"context"
	"fmt"
	"github.com/pshvedko/nocopy/broker2/message"
	"log/slog"

	"github.com/google/uuid"
)

type Subscription interface {
	Unsubscribe() error
}

type Transport interface {
	Flush() error
	Subscribe(at string, f func(string, message.Message)) (Subscription, error)
	QueueSubscribe(at string, queue string, f func(string, message.Message)) (Subscription, error)
}

type Exchange struct {
	transport     Transport
	topic         [2][]string
	subscriptions []Subscription
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

func (e *Exchange) Listen(ctx context.Context, at string, to ...string) error {
	a := fmt.Sprint("@", at)
	u := fmt.Sprint("#", at)

	for {
		s, err := e.transport.Subscribe(a, e.route)
		if err != nil {
			return err
		}
		slog.Info("LISTEN", "at", a)

		e.subscriptions = append(e.subscriptions, s)

		s, err = e.transport.QueueSubscribe(u, at, e.route)
		if err != nil {
			return err
		}
		slog.Info("LISTEN", "at", u)

		e.subscriptions = append(e.subscriptions, s)

		e.topic[0] = append(e.topic[0], a)
		e.topic[1] = append(e.topic[1], u)

		if len(to) == 0 {
			break
		}

		a = fmt.Sprint(a, ".", to[0])
		u = fmt.Sprint(u, ".", to[0])

		to = to[1:]
	}

	return e.transport.Flush()
}

func (e *Exchange) Finish() {
	//TODO implement me
	panic("implement me")
}

func (e *Exchange) Shutdown() {
	//TODO implement me
	panic("implement me")
}

func (e *Exchange) route(topic string, message message.Message) {

}
