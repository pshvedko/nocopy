package message

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
)

var (
	ErrRedundantMessage = errors.New("redundant message")
	ErrNoPayload        = errors.New("no payload")
	ErrEmpty            = errors.New("empty")
)

type Address interface {
	ID() uuid.UUID
	From() string
	Return() []string
	To() string
	Type() int
	Handle() string
}

type Message interface {
	Address
}

type Middleware interface {
	Decode(context.Context, []byte) (context.Context, error)
}

type MiddlewareFunc func(context.Context, []byte) (context.Context, error)

func (f MiddlewareFunc) Decode(ctx context.Context, bytes []byte) (context.Context, error) {
	return f(ctx, bytes)
}

type Decoder interface {
	Decode(context.Context, string, []byte, ...Middleware) (context.Context, Message, error)
}

type Envelope struct {
	ID     uuid.UUID `json:"id,omitempty"`
	From   string    `json:"from,omitempty"`
	Return []string  `json:"return,omitempty"`
	To     string    `json:"to,omitempty"`
	Type   int       `json:"type,omitempty"`
	Handle string    `json:"handle,omitempty"`
}

type Raw struct {
	Envelope
	Payload json.RawMessage
}

func (r Raw) ID() uuid.UUID {
	return r.Envelope.ID
}

func (r Raw) From() string {
	return r.Envelope.From
}

func (r Raw) Return() []string {
	return r.Envelope.Return
}

func (r Raw) To() string {
	return r.Envelope.To
}

func (r Raw) Type() int {
	return r.Envelope.Type
}

func (r Raw) Handle() string {
	return r.Envelope.Handle
}

type UnmarshalFunc func([]byte) error

func (f UnmarshalFunc) UnmarshalJSON(bytes []byte) error { return f(bytes) }

func Decode(ctx context.Context, topic string, bytes []byte, middlewares ...Middleware) (context.Context, Message, error) {
	var i int
	var r Raw

	decoders := append(make([]UnmarshalFunc, 0, 3+len(middlewares)),
		func(bytes []byte) error {
			i++
			return json.Unmarshal(bytes, &r.Envelope)
		},
	)

	for _, middleware := range middlewares {
		decoders = append(decoders,
			func(bytes []byte) (err error) {
				i++
				ctx, err = middleware.Decode(ctx, bytes)
				return
			},
		)
	}

	decoders = append(decoders,
		func(bytes []byte) error {
			i++
			return json.Unmarshal(bytes, &r.Payload)
		},
		func(bytes []byte) error {
			i++
			return ErrRedundantMessage
		},
	)

	err := json.Unmarshal(bytes, &decoders)
	if err != nil {
		return nil, nil, err
	}

	switch i {
	case 0:
		return nil, nil, ErrEmpty
	case 1 + len(middlewares):
		return nil, nil, ErrNoPayload
	}

	return ctx, r, nil
}

type DecodeFunc func(context.Context, string, []byte, ...Middleware) (context.Context, Message, error)

func (f DecodeFunc) Decode(ctx context.Context, topic string, bytes []byte, middlewares ...Middleware) (context.Context, Message, error) {
	return f(ctx, topic, bytes, middlewares...)
}

type Body interface {
	Encode() ([]byte, error)
}

type content[T any] struct {
	a T
}

func (c content[T]) Encode() ([]byte, error) {
	return json.Marshal(c.a)
}

func Content[T any](a T) Body {
	return content[T]{
		a: a,
	}
}

type Handler func(context.Context, Message) (Body, error)
type Catcher func(context.Context, Message)
