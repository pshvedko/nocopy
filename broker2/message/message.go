package message

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
)

var (
	ErrRedundantMessage = errors.New("redundant message")
	ErrIllegalType      = errors.New("illegal type")
	ErrNoPayload        = errors.New("no payload")
	ErrEmpty            = errors.New("empty")
)

type Type int

const (
	Query Type = iota
	Reply
	Failure
	Broadcast
)

type Address interface {
	ID() uuid.UUID
	From() string
	Return() []string
	To() string
	Type() Type
	Handle() string
}

type Message interface {
	Address
}

type MiddlewareFunc func(context.Context, []byte) (context.Context, error)

type Mediator interface {
	Middleware(string) []MiddlewareFunc
}

type Decoder interface {
	Decode(context.Context, []byte, Mediator) (context.Context, Message, error)
}

type Envelope struct {
	ID     uuid.UUID `json:"id,omitempty"`
	From   string    `json:"from,omitempty"`
	Return []string  `json:"return,omitempty"`
	To     string    `json:"to,omitempty"`
	Type   Type      `json:"type,omitempty"`
	Method string    `json:"method,omitempty"`
}

type Error struct {
	Code int    `json:"code,omitempty"`
	Text string `json:"text,omitempty"`
}

func (e Error) Error() string {
	return e.Text
}

type Raw struct {
	Error
	Envelope
	RawMessage json.RawMessage
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

func (r Raw) Type() Type {
	return r.Envelope.Type
}

func (r Raw) Handle() string {
	return r.Envelope.Method
}

type UnmarshalFunc func([]byte) error

func (f UnmarshalFunc) UnmarshalJSON(bytes []byte) error { return f(bytes) }

func Decode(ctx context.Context, bytes []byte, mediator Mediator) (context.Context, Message, error) {
	var i int
	var u int
	var v interface{}
	var r Raw
	var uu []UnmarshalFunc

	uu = append(uu, func(bytes []byte) error {
		v = &r.Error
		i++
		err := json.Unmarshal(bytes, &r.Envelope)
		if err != nil {
			return err
		}
		switch r.Envelope.Type {
		case Query, Broadcast:
			for _, w := range mediator.Middleware(r.Envelope.Method) {
				uu = append(uu,
					func(bytes []byte) (err error) {
						i++
						ctx, err = w(ctx, bytes)
						return
					},
				)
				u++
			}
			fallthrough
		case Reply:
			v = &r.RawMessage
			fallthrough
		case Failure:
			uu = append(uu,
				func(bytes []byte) error {
					i++
					return json.Unmarshal(bytes, v)
				},
				func(bytes []byte) error {
					i++
					return ErrRedundantMessage
				},
			)
		default:
			return ErrIllegalType
		}
		return nil
	})

	err := json.Unmarshal(bytes, &uu)
	if err != nil {
		return nil, nil, err
	}

	switch i {
	case 0:
		return nil, nil, ErrEmpty
	case 1 + u:
		return nil, nil, ErrNoPayload
	}

	return ctx, r, nil
}

type DecodeFunc func(context.Context, []byte, Mediator) (context.Context, Message, error)

func (f DecodeFunc) Decode(ctx context.Context, bytes []byte, mediator Mediator) (context.Context, Message, error) {
	return f(ctx, bytes, mediator)
}

type Body interface {
	Encode() ([]byte, error)
}

type Content[T any] struct {
	a T
}

func (c Content[T]) Encode() ([]byte, error) {
	return json.Marshal(c.a)
}

func NewContent[T any](a T) Body {
	return Content[T]{
		a: a,
	}
}

func NewFailure[T any](code int, err T) error {
	return Error{
		Code: code,
		Text: fmt.Sprint(err),
	}
}

type Handler func(context.Context, Message) (Body, error)
type Catcher func(context.Context, Message)
