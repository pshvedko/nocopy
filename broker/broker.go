package broker

import (
	"context"
	"errors"
	"net/url"

	"github.com/google/uuid"

	"github.com/pshvedko/nocopy/broker/message"
	"github.com/pshvedko/nocopy/broker/nats"
)

type Broker interface {
	At(int) string
	Handle(string, message.Handler)
	Catch(string, message.Catcher)
	Message(context.Context, string, string, any, ...any) (uuid.UUID, error)
	Send(context.Context, message.Message, ...any) (uuid.UUID, error)
	Listen(context.Context, string, string, string) error
	Finish()
	Shutdown()
}

func New(name string) (Broker, error) {
	ur1, err := url.Parse(name)
	if err != nil {
		return nil, err
	}
	switch ur1.Scheme {
	case "nats":
		return nats.New(ur1)
	default:
		return nil, errors.New("invalid broker scheme")
	}
}
