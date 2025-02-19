package exchange

import (
	"github.com/google/uuid"

	"github.com/pshvedko/nocopy/broker2/message"
)

type Option interface {
	mustBeInternalExchangeOptionMatherFucker()
}

type OptionWithFrom struct {
	n int
}

func (o OptionWithFrom) mustBeInternalExchangeOptionMatherFucker() {}

func WithFrom(n int) Option { return OptionWithFrom{n: n} }

func WithMaxFrom() Option { return WithFrom(-1) }

func WithMinFrom() Option { return WithFrom(0) }

type OptionWithID struct {
	id uuid.UUID
}

func (o OptionWithID) mustBeInternalExchangeOptionMatherFucker() {}

func WithID(id uuid.UUID) Option { return OptionWithID{id: id} }

func (e *Exchange) Apply(m message.Message, options ...Option) message.Message {
	b := message.NewMessage(m)

	for _, option := range options {
		switch o := option.(type) {
		case OptionWithID:
			b = b.WithID(o.id)
		case OptionWithFrom:
			_, f := e.Topic(o.n)
			b = b.WithFrom(f)
		}
	}

	return b.Build()
}
