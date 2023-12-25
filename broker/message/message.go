package message

import (
	"errors"

	"github.com/google/uuid"
)

type ID = uuid.UUID
type RE = []string
type BY = string

type Query interface {
	ID() ID
	RE() RE
	BY() BY
	Unmarshal(any) error
}

type Reply interface {
	Marshal() ([]byte, error)
}

var ErrOption = errors.New("invalid query option")

type Option struct {
	ID func() ID
	RE func() RE
}

func Options(oo []any) (o Option, err error) {
	for _, v := range append([]any{uuid.New, func() RE { return nil }}, oo...) {
		switch x := v.(type) {
		case func() ID:
			o.ID = x
		case ID:
			o.ID = func() uuid.UUID { return x }
		case func() RE:
			o.RE = x
		case RE:
			o.RE = func() []string { return x }
		case Query:
			o.ID = x.ID
			o.RE = x.RE
		default:
			err = ErrOption
			return
		}
	}
	return
}
