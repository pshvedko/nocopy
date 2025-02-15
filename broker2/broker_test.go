package exchange

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/pshvedko/nocopy/broker2/message"
)

func Service(ctx context.Context, url, name string, topic ...string) error {
	b, err := New(url)
	if err != nil {
		return err
	}
	defer b.Shutdown()
	b.Handle("echo", HandleEcho)
	err = b.Listen(ctx, name, topic...)
	if err != nil {
		return err
	}
	slog.Info("READY")
	<-ctx.Done()
	b.Finish()
	slog.Info("DONE")
	return nil
}

func HandleEcho(ctx context.Context, m message.Message) (message.Body, error) {
	slog.Info("++READ", "id", m.ID(), "from", m.From(), "to", m.To(), "method", m.Method(), "type", m.Type(), "return", m.Return())
	return m, nil
}

type Hello struct {
	Name string `json:"name,omitempty"`
}

func TestNewTransport(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.TODO(), 1*time.Minute)

	var wg sync.WaitGroup
	defer wg.Wait()
	defer cancel()

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := Service(ctx, "nats://nats", "1", "1", "1")
		require.NoError(t, err)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := Service(ctx, "nats://nats", "1", "1", "2")
		require.NoError(t, err)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := Service(ctx, "nats://nats", "1", "2", "1")
		require.NoError(t, err)
	}()

	echo := make(chan message.Message, 1)
	b, err := New("nats://nats")
	require.NoError(t, err)
	defer b.Shutdown()
	b.Catch("echo", func(ctx context.Context, m message.Message) {
		echo <- m
	})
	err = b.Listen(ctx, "0", "0", "0")
	require.NoError(t, err)
	defer b.Finish()

	_, err = b.Send(ctx, message.New(message.Empty{}).
		WithTo("1").
		WithFrom("0").
		WithID(uuid.New()).
		WithMethod("echo").
		WithBody(message.NewBody(Hello{Name: "alice"})))
	require.NoError(t, err)
	m := <-echo
	slog.Info("--READ", "id", m.ID(), "from", m.From(), "to", m.To(), "method", m.Method(), "type", m.Type(), "return", m.Return())

}
