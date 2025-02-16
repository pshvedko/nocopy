package exchange

import (
	"context"
	"github.com/pshvedko/nocopy/broker2/exchange"
	"github.com/stretchr/testify/require"
	"log/slog"
	"os"
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
	out := slog.With("id", m.ID(), "by", m.Method(), "at", m.From(), "to", m.To(), "type", m.Type())
	err := t.Transport.Publish(ctx, m, mediator)
	if err != nil {
		out = out.With("error", err)
	}
	out.Info("SEND->")
	return err
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

	for i, at := range [][]string{
		{"service", "one", "one"},
		{"service", "one", "two"},
		{"service", "two", "one"},
		{"service", "two", "two"},
	} {
		b, err := s.NewService(i, at[0], at[1:]...)
		require.NoError(t, err)
		bb = append(bb, b)
	}

	t.Run("Send", s.TestSend)

	for _, b := range bb {
		b.Shutdown()
	}
}

type Echo struct {
	Text string
}

type Empty struct {
	Number int
}

func (s Suit) Message(ctx context.Context, m message.Message) {
	s.messages <- m
}

func (s Suit) NewService(i int, name string, topic ...string) (Broker, error) {
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
		var e Echo
		err := m.Decode(&e)
		if err != nil {
			return nil, err
		}
		return message.NewBody(Echo{Text: "Hello, " + e.Text + "!"}), nil
	})
	b.Handle("empty", func(ctx context.Context, m message.Message) (message.Body, error) {
		var e Empty
		err := m.Decode(&e)
		if err != nil {
			return nil, err
		}
		return message.NewBody(Echo{Text: "How are you?"}), nil
	})
	b.Handle("error", func(ctx context.Context, m message.Message) (message.Body, error) {
		var e Empty
		err := m.Decode(&e)
		if err != nil {
			return nil, err
		}
		switch e.Number {
		case 1:
			return nil, message.NewError(400, os.ErrInvalid)
		default:
			return nil, os.ErrClosed
		}
	})
	b.Handle("number", func(ctx context.Context, m message.Message) (message.Body, error) {
		var e Empty
		err := m.Decode(&e)
		if err != nil {
			return nil, err
		}
		return message.NewBody(Empty{Number: i}), nil
	})
	b.Wrap(LogTransport{Transport: b.Transport()})
	err = b.Listen(s.ctx, name, topic...)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (s Suit) TestSend(t *testing.T) {
	b, err := New(s.url)
	require.NoError(t, err)
	defer b.Shutdown()
	b.Catch("echo", s.Message)
	b.Catch("hello", s.Message)
	b.Catch("empty", s.Message)
	b.Catch("error", s.Message)
	b.Catch("number", s.Message)
	err = b.Listen(s.ctx, "client", "zero", "zero")
	require.NoError(t, err)
	defer b.Finish()

	var e Echo
	_, err = b.Send(s.ctx, message.New().
		WithTo("service").
		WithFrom("client").
		WithMethod("echo").
		WithBody(message.NewBody(Echo{Text: "Alice"})))
	require.NoError(t, err)
	m := <-s.messages
	err = m.Decode(&e)
	require.NoError(t, err)
	require.Equal(t, Echo{Text: "Alice"}, e)

	_, err = b.Send(s.ctx, message.New().
		WithTo("service").
		WithFrom("client").
		WithMethod("hello").
		WithBody(message.NewBody(Echo{Text: "Alice"})))
	require.NoError(t, err)
	m = <-s.messages
	err = m.Decode(&e)
	require.NoError(t, err)
	require.Equal(t, Echo{Text: "Hello, Alice!"}, e)

	_, err = b.Send(s.ctx, message.New().
		WithTo("service").
		WithFrom("client").
		WithMethod("empty").
		WithBody(nil))
	require.NoError(t, err)
	m = <-s.messages
	err = m.Decode(&e)
	require.NoError(t, err)
	require.Equal(t, Echo{Text: "How are you?"}, e)

	_, err = b.Send(s.ctx, message.New().
		WithTo("service").
		WithFrom("client").
		WithMethod("error").
		WithBody(message.NewBody(Empty{})))
	require.NoError(t, err)
	m = <-s.messages
	err = m.Decode(&e)
	require.Error(t, err)
	require.ErrorIs(t, err, message.Error{
		Code: 500,
		Text: os.ErrClosed.Error(),
	})

	_, err = b.Send(s.ctx, message.New().
		WithTo("service").
		WithFrom("client").
		WithMethod("error").
		WithBody(message.NewBody(Empty{Number: 1})))
	require.NoError(t, err)
	m = <-s.messages
	err = m.Decode(&e)
	require.Error(t, err)
	require.ErrorIs(t, err, message.Error{
		Code: 400,
		Text: os.ErrInvalid.Error(),
	})

	var n Empty
	_, err = b.Send(s.ctx, message.New().
		WithTo("service.two.two").
		WithFrom("client").
		WithMethod("number").
		WithBody(message.NewBody(Empty{Number: 1})))
	require.NoError(t, err)
	m = <-s.messages
	err = m.Decode(&n)
	require.NoError(t, err)
	require.Equal(t, 3, n.Number)
}
