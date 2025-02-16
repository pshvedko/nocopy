package exchange

import (
	"github.com/google/uuid"
	"github.com/pshvedko/nocopy/broker2/message"
)

type Exchanger interface {
	Topic(n int) (int, string)
}

type Option func(Exchanger, message.Builder) message.Builder

func WithFrom(n int) Option {
	return func(e Exchanger, b message.Builder) message.Builder {
		_, f := e.Topic(n)
		return b.WithFrom(f)
	}
}

func WithMaxFrom() Option {
	return WithFrom(-1)
}

func WithMinFrom() Option {
	return WithFrom(0)
}

func WithID(id uuid.UUID) Option {
	return func(e Exchanger, b message.Builder) message.Builder {
		return b.WithID(id)
	}
}

func (e *Exchange) Apply(m message.Message, options ...Option) message.Message {
	b := message.NewMessage(m)
	for _, option := range options {
		b = option(e, b)
	}
	return b.Build()
}
