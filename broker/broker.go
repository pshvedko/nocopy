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
	Handle(string, message.Handler)
	Catch(string, message.Catcher)
	Send(context.Context, string, string, any, ...any) (uuid.UUID, error)
	Listen(context.Context, string, string, string) error
	Finish()
	Shutdown()
	At(int) string
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
