package exchange

import (
	"context"
	"errors"
	"net/url"

	"github.com/google/uuid"

	"github.com/pshvedko/nocopy/broker2/exchange"
	"github.com/pshvedko/nocopy/broker2/message"
	"github.com/pshvedko/nocopy/broker2/transport/nats"
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
	t, err := NewTransport(name, message.Decode)
	if err != nil {
		return nil, err
	}
	return exchange.New(t), nil
}

func NewTransport(name string, f message.DecodeFunc) (exchange.Transport, error) {
	u, err := url.Parse(name)
	if err != nil {
		return nil, err
	}
	switch u.Scheme {
	case "nats":
		return nats.New(u, f)
	default:
		return nil, errors.New("invalid broker scheme")
	}
}
