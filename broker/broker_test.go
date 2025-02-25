package broker

import (
	"context"
	"github.com/pshvedko/nocopy/internal/log"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/pshvedko/nocopy/broker/exchange"
	"github.com/pshvedko/nocopy/broker/message"
)

type Suit struct {
	url      string
	ctx      context.Context
	messages chan message.Message
	group    sync.WaitGroup
}

type L struct {
	t testing.TB
}

func (w L) Write(p []byte) (n int, err error) {
	w.t.Logf("%s", p)
	return 0, err
}

func NewLogger(t *testing.T) *slog.Logger {
	return slog.New(slog.NewTextHandler(L{t: t}, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestExchange(t *testing.T) {
	ur1 := os.Getenv("TEST_NATS")
	if ur1 == "" {
		t.SkipNow()
	}
	ctx, cancel := context.WithTimeout(context.TODO(), 1*time.Minute)
	defer cancel()
	s := Suit{
		url:      ur1,
		ctx:      ctx,
		messages: make(chan message.Message, 1),
	}
	slog.SetDefault(NewLogger(t))
	t.Run("Service", s.TestService)
}

func (s *Suit) TestService(t *testing.T) {
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

	t.Run("Query", s.TestQuery)

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

func (s *Suit) Message(ctx context.Context, m message.Message) bool {
	select {
	case s.messages <- m:
		return true
	case <-ctx.Done():
		return false
	}
}

func (s *Suit) NewService(i int, name string, topic ...string) (Broker, error) {
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
	b.Handle("hello2", func(ctx context.Context, m message.Message) (message.Body, error) {
		var e Echo
		err := m.Decode(&e)
		if err != nil {
			return nil, err
		}
		return message.NewBody(Echo{Text: "Hello, " + e.Text + "!!"}), nil
	})
	b.Handle("empty", func(ctx context.Context, m message.Message) (message.Body, error) {
		var e Empty
		err := m.Decode(&e)
		if err != nil {
			return nil, err
		}
		_, _ = b.Answer(ctx, m, message.NewBody(Echo{Text: "How are you?"}), exchange.WithMaxFrom())
		return nil, nil
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
		if m.Type() == message.Broadcast {
			defer s.group.Done()
		}
		var e Empty
		err := m.Decode(&e)
		if err != nil {
			return nil, err
		}
		return message.NewBody(Empty{Number: i}), nil
	})
	b.UseTransport(log.Transport{Transport: b.Transport()})
	err = b.Listen(s.ctx, name, topic...)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (s *Suit) TestQuery(t *testing.T) {
	b, err := New(s.url)
	require.NoError(t, err)
	defer b.Shutdown()
	b.Catch("echo", s.Message)
	b.Catch("hello", s.Message)
	b.Catch("empty", s.Message)
	b.Catch("error", s.Message)
	b.Catch("number", s.Message)
	b.UseTransport(log.Transport{Transport: b.Transport()})
	err = b.Listen(s.ctx, "client", "zero", "zero")
	require.NoError(t, err)
	defer b.Finish()

	var e Echo
	id, err := b.Send(s.ctx, message.New().
		WithTo("service").
		WithFrom("client").
		WithMethod("echo").
		WithBody(message.NewBody(Echo{Text: "Alice"})), exchange.WithMinFrom(), exchange.WithMaxFrom())
	require.NoError(t, err)
	m := <-s.messages
	err = m.Decode(&e)
	require.NoError(t, err)
	require.Equal(t, Echo{Text: "Alice"}, e)

	_, err = b.Send(s.ctx, message.New().
		WithTo("service").
		WithFrom("client").
		WithMethod("hello").
		WithBody(message.NewBody(Echo{Text: "Alice"})), exchange.WithID(id))
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

	s.group.Add(2)
	_, err = b.Send(s.ctx, message.New().
		WithType(message.Broadcast).
		WithTo("service.two").
		WithFrom("client").
		WithMethod("number").
		WithBody(message.NewBody(Empty{Number: 7})))
	require.NoError(t, err)
	s.group.Wait()

	m, err = b.Request(s.ctx, "service", "hello", message.NewBody(Echo{Text: "Bob"}))
	require.NoError(t, err)
	err = m.Decode(&e)
	require.NoError(t, err)
	require.Equal(t, Echo{Text: "Hello, Bob!"}, e)

	m, err = b.Request(s.ctx, "service", "hello2", message.NewBody(Echo{Text: "Bob"}))
	require.NoError(t, err)
	err = m.Decode(&e)
	require.NoError(t, err)
	require.Equal(t, Echo{Text: "Hello, Bob!!"}, e)

	m, err = b.Request(s.ctx, "service", "error", message.NewBody(Empty{}))
	require.NoError(t, err)
	err = m.Decode(&e)
	require.Error(t, err)
	require.ErrorIs(t, err, message.Error{
		Code: 500,
		Text: os.ErrClosed.Error(),
	})

}
