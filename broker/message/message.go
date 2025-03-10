package message

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
)

var (
	ErrRedundantMessage = errors.New("redundant message")
	ErrIllegalType      = errors.New("illegal type")
	ErrNoPayload        = errors.New("no payload")
	ErrEmpty            = errors.New("empty")
	ErrIllegalID        = errors.New("illegal id")
)

type Type int

const (
	Query Type = iota
	Answer
	Request
	Failure
	Broadcast
)

type Address interface {
	ID() uuid.UUID
	From() string
	Return() []string
	To() string
	Type() Type
	Method() string
}

type Message interface {
	Address
	Body
	Decode(any) error
}

type DecodeFunc func(context.Context, []byte) (context.Context, error)

func (f DecodeFunc) Decode(ctx context.Context, bytes []byte) (context.Context, error) {
	return f(ctx, bytes)
}

type EncodeFunc func(context.Context) ([]byte, error)

func (f EncodeFunc) Encode(ctx context.Context) ([]byte, error) {
	return f(ctx)
}

type FormatFunc struct {
	DecodeFunc
	EncodeFunc
}

type Middleware interface {
	Decode(ctx context.Context, bytes []byte) (context.Context, error)
	Encode(ctx context.Context) ([]byte, error)
}

type Mediator interface {
	Middleware(string) []Middleware
}

type Encoder interface {
	Encode(context.Context, Message) ([]byte, error)
}

type Decoder interface {
	Decode(context.Context, []byte) (context.Context, Message, error)
	Processor
}

type Processor interface {
	Do(context.Context, Message)
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

func (e Error) Encode() ([]byte, error) {
	return json.Marshal(e)
}

func (e Error) Error() string {
	return e.Text
}

type Raw struct {
	Err      Error
	Envelope Envelope
	Body     json.RawMessage
}

func (r Raw) Decode(v any) error {
	if r.Envelope.Type == Failure {
		return r.Err
	}
	return json.Unmarshal(r.Body, v)
}

func (r Raw) Encode() ([]byte, error) {
	if r.Envelope.Type == Failure {
		return r.Err.Encode()
	}
	return r.Body.MarshalJSON()
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

func (r Raw) Method() string {
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
		v = &r.Err
		i++
		err := json.Unmarshal(bytes, &r.Envelope)
		if err != nil {
			return err
		}
		switch r.Envelope.Type {
		case Query, Request, Broadcast:
			for _, w := range mediator.Middleware(r.Envelope.Method) {
				uu = append(uu,
					func(bytes []byte) (err error) {
						i++
						ctx, err = w.Decode(ctx, bytes)
						return
					},
				)
				u++
			}
			fallthrough
		case Answer:
			v = &r.Body
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

type MarshalFunc func() ([]byte, error)

func (f MarshalFunc) MarshalJSON() ([]byte, error) { return f() }

func Encode(ctx context.Context, m Message, mediator Mediator) ([]byte, error) {
	if m.ID() == uuid.Nil {
		return nil, ErrEmpty
	}

	var ww []Middleware
	if m.Type()&Answer == Query {
		ww = mediator.Middleware(m.Method())
	}

	mm := append(make([]MarshalFunc, 0, 2+len(ww)), func() ([]byte, error) {
		return json.Marshal(Envelope{
			ID:     m.ID(),
			From:   m.From(),
			Return: m.Return(),
			To:     m.To(),
			Type:   m.Type(),
			Method: m.Method(),
		})
	})

	for _, w := range ww {
		mm = append(mm, func() ([]byte, error) {
			return w.Encode(ctx)
		})
	}

	mm = append(mm, func() ([]byte, error) {
		return m.Encode()
	})

	return json.Marshal(mm)
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

func NewBody[T any](a T) Body {
	return Content[T]{
		a: a,
	}
}

func NewErrorString(code int, text string) Error {
	return Error{
		Code: code,
		Text: text,
	}
}

func NewError(code int, err error) Error { return NewErrorString(code, err.Error()) }

type HandleFunc func(context.Context, Message) (Body, error)
type CatchFunc func(context.Context, Message) bool

type Empty struct {
	id uuid.UUID
}

func (e Empty) ID() uuid.UUID { return e.id }

func (e Empty) From() string { return "" }

func (e Empty) Return() []string { return []string{} }

func (e Empty) To() string { return "" }

func (e Empty) Type() Type { return 0 }

func (e Empty) Method() string { return "" }

func (e Empty) Encode() ([]byte, error) { return []byte{'n', 'u', 'l', 'l'}, nil }

func (e Empty) Decode(any) error { return nil }
