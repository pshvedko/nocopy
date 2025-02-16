package exchange

import (
	"context"
	"github.com/pshvedko/nocopy/broker2/exchange"
	"github.com/stretchr/testify/require"
	"log/slog"
	"testing"
	"time"

	"github.com/pshvedko/nocopy/broker2/message"
)

type LogTransport struct {
	exchange.Transport
}

func (t LogTransport) Subscribe(ctx context.Context, at string, mediator message.Mediator, doer exchange.Doer) (exchange.Subscription, error) {
	slog.Info("LISTEN", "at", at, "wide", true)
	return t.Transport.Subscribe(ctx, at, mediator, doer)
}

type LogInput struct {
	exchange.Doer
}

func (l LogInput) Do(ctx context.Context, m message.Message) {
	slog.Info("<-READ", "id", m.ID(), "by", m.Method(), "at", m.To(), "from", m.From(), "type", m.Type())
	l.Doer.Do(ctx, m)
}

func (t LogTransport) QueueSubscribe(ctx context.Context, at string, by string, mediator message.Mediator, doer exchange.Doer) (exchange.Subscription, error) {
	slog.Info("LISTEN", "at", at)
	return t.Transport.QueueSubscribe(ctx, at, by, mediator, LogInput{Doer: doer})
}

func (t LogTransport) Publish(ctx context.Context, m message.Message, mediator message.Mediator) error {
	slog.Info("SEND->", "id", m.ID(), "by", m.Method(), "at", m.From(), "to", m.To(), "type", m.Type())
	return t.Transport.Publish(ctx, m, mediator)
}

func (t LogTransport) Unsubscribe(topic exchange.Topic) error {
	switch topic.Wide() {
	case true:
		slog.Info("FINISH", "at", topic, "wide", topic.Wide())
	default:
		slog.Info("FINISH", "at", topic)
	}
	return t.Transport.Unsubscribe(topic)
}

type Suit struct {
	url      string
	ctx      context.Context
	messages chan message.Message
}

func (s Suit) Message(ctx context.Context, m message.Message) {
	s.messages <- m
}

func (s Suit) NewService(name string, topic ...string) (Broker, error) {
	b, err := New(s.url)
	if err != nil {
		return nil, err
	}
	b.Handle("echo", func(ctx context.Context, m message.Message) (message.Body, error) {
		var e Echo
		err := m.Decode(&e)
		if err != nil {
			return nil, err
		}
		return m, nil
	})
	b.Handle("hello", func(ctx context.Context, m message.Message) (message.Body, error) {
		var h Hello
		err := m.Decode(&h)
		if err != nil {
			return nil, err
		}
		return message.NewBody(Hello{Name: "Hello, " + h.Name + "!"}), nil
	})
	b.Wrap(LogTransport{Transport: b.Transport()})
	err = b.Listen(s.ctx, name, topic...)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func TestExchange(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.TODO(), 1*time.Minute)
	defer cancel()
	s := Suit{
		url:      "nats://nats",
		ctx:      ctx,
		messages: make(chan message.Message, 1),
	}
	t.Run("Service", s.TestService)
}

func (s Suit) TestService(t *testing.T) {
	var bb []Broker

	for _, at := range [][]string{
		{"service", "one", "one"},
		{"service", "one", "two"},
		{"service", "two", "one"},
	} {
		b, err := s.NewService(at[0], at[1:]...)
		require.NoError(t, err)
		bb = append(bb, b)
	}

	t.Run("Request", s.TestRequest)

	for _, b := range bb {
		b.Shutdown()
	}
}

type Echo struct {
	Name string
}

type Hello struct {
	Name string
}

func (s Suit) TestRequest(t *testing.T) {
	b, err := New(s.url)
	require.NoError(t, err)
	defer b.Shutdown()
	b.Catch("echo", s.Message)
	b.Catch("hello", s.Message)
	err = b.Listen(s.ctx, "client", "zero", "zero")
	require.NoError(t, err)
	defer b.Finish()

	var echo Echo
	_, err = b.Send(s.ctx, message.New().
		WithTo("service").
		WithFrom("client").
		WithMethod("echo").
		WithBody(message.NewBody(Echo{Name: "Alice"})))
	require.NoError(t, err)
	m := <-s.messages
	err = m.Decode(&echo)
	require.NoError(t, err)
	require.Equal(t, Echo{Name: "Alice"}, echo)

	var hello Hello
	_, err = b.Send(s.ctx, message.New().
		WithTo("service").
		WithFrom("client").
		WithMethod("hello").
		WithBody(message.NewBody(Hello{Name: "Alice"})))
	require.NoError(t, err)
	m = <-s.messages
	err = m.Decode(&hello)
	require.NoError(t, err)
	require.Equal(t, Hello{Name: "Hello, Alice!"}, hello)

}
