package message

import (
	"context"
	"encoding/json"
)

type Message interface {
}

type Encoder interface {
	Encode() ([]byte, error)
}

type Body interface {
	Encoder
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
