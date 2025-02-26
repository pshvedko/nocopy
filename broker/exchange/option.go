package exchange

import (
	"github.com/google/uuid"
	"time"

	"github.com/pshvedko/nocopy/broker/message"
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

type Config struct {
	Timeout time.Duration
}

type OptionWithTimeout struct {
	timeout time.Duration
}

func (o OptionWithTimeout) mustBeInternalExchangeOptionMatherFucker() {}

func WithTimeout(timeout time.Duration) Option { return OptionWithTimeout{timeout: timeout} }

func (e *Exchange) Apply(m message.Message, options ...Option) (Config, message.Message) {
	c := e.config
	b := message.NewMessage(m)

	for _, option := range options {
		switch o := option.(type) {
		case OptionWithID:
			b = b.WithID(o.id)
		case OptionWithFrom:
			_, f := e.Topic(o.n)
			b = b.WithFrom(f)
		case OptionWithTimeout:
			c.Timeout = o.timeout
		}
	}

	return c, b.Build()
}
