package broker

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"net/url"

	"github.com/pshvedko/nocopy/broker/message"
	"github.com/pshvedko/nocopy/broker/nats"
)

type Broker interface {
	Handle(string, func(context.Context, message.Query) (message.Reply, error))
	Query(context.Context, string, string, any, ...any) (uuid.UUID, error)
	Listen(context.Context, string, string, string) error
	Shutdown(context.Context) error
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
