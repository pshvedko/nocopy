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
	Handle(string, message.HandleFunc)
	Catch(string, message.CatchFunc)
	Use(message.Middleware)
	Message(context.Context, string, string, message.Body, ...exchange.Option) (uuid.UUID, error)
	Request(context.Context, string, string, message.Body, ...exchange.Option) (uuid.UUID, message.Message, error)
	Answer(context.Context, message.Message, message.Body, ...exchange.Option) (uuid.UUID, error)
	Send(context.Context, message.Message, ...exchange.Option) (uuid.UUID, error)
	Listen(context.Context, string, ...string) error
	Topic(int) (int, string)
	Finish()
	Shutdown()
	Transport() exchange.Transport
	Wrap(exchange.Transport)
	MaxRequestTopic(bool)
	MaxRespondTopic(bool)
}

func New(name string) (Broker, error) {
	t, err := NewTransport(name)
	if err != nil {
		return nil, err
	}
	return exchange.New(t), nil
}

func NewTransport(name string) (exchange.Transport, error) {
	u, err := url.Parse(name)
	if err != nil {
		return nil, err
	}
	switch u.Scheme {
	case "nats":
		return nats.New(u)
	default:
		return nil, errors.New("invalid broker scheme")
	}
}
