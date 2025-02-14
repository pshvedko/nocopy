package exchange

import (
	"context"
	"errors"
	"net/url"

	"github.com/google/uuid"

	"github.com/pshvedko/nocopy/broker2/exchange/message"
	"github.com/pshvedko/nocopy/broker2/exchange/transport/nats"
)

type Broker interface {
	Handle(string, message.Handler)
	Catch(string, message.Catcher)
	Message(context.Context, string, string, message.Body, ...any) (uuid.UUID, error)
	Request(context.Context, string, string, message.Body, ...any) (uuid.UUID, message.Message, error)
	Send(context.Context, message.Message, ...any) (uuid.UUID, error)
	Listen(context.Context, string, ...string) error
	Finish()
	Shutdown()
}

func New(name string) (Broker, error) {
	u, err := url.Parse(name)
	if err != nil {
		return nil, err
	}
	t, err := NewTransport(*u)
	if err != nil {
		return nil, err
	}
	return NewExchange(t), nil
}

func NewTransport(u url.URL) (Transport, error) {
	switch u.Scheme {
	case "nats":
		return nats.New(u)
	default:
		return nil, errors.New("invalid broker scheme")
	}
}
