package broker

import (
	"context"
	"errors"
	"net/url"

	"github.com/google/uuid"

	"github.com/pshvedko/nocopy/broker/exchange"
	"github.com/pshvedko/nocopy/broker/message"
	"github.com/pshvedko/nocopy/broker/transport/nats"
)

type Broker interface {
	Handle(string, message.HandleFunc)
	Catch(string, message.CatchFunc)
	Message(context.Context, string, string, message.Body, ...exchange.Option) (uuid.UUID, error)
	Request(context.Context, string, string, message.Body, ...exchange.Option) (message.Message, error)
	Answer(context.Context, message.Message, message.Body, ...exchange.Option) (uuid.UUID, error)
	Forward(context.Context, string, message.Message, ...exchange.Option) (uuid.UUID, error)
	Backward(context.Context, message.Message, ...exchange.Option) (uuid.UUID, error)
	Send(context.Context, message.Message, ...exchange.Option) (uuid.UUID, error)
	Listen(context.Context, string, ...string) error
	Topic(int) (int, string)
	Finish()
	Shutdown()
	Transport() exchange.Transport // TODO move to transport
	UseTransport(exchange.Transport)
	UseMiddleware(...message.Middleware)
	UseOptions(...exchange.Option)
}

func New(name, ident string) (Broker, error) {
	t, err := NewTransport(name, ident)
	if err != nil {
		return nil, err
	}
	return exchange.New(t), nil
}

func NewTransport(name string, ident string) (exchange.Transport, error) {
	u, err := url.Parse(name)
	if err != nil {
		return nil, err
	}
	switch u.Scheme {
	case "nats":
		return nats.New(u, ident)
	default:
		return nil, errors.New("invalid broker scheme")
	}
}
