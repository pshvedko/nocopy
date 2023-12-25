package message

import (
	"errors"

	"github.com/google/uuid"
)

type Query interface {
	ID() uuid.UUID
	RE() []string
	BY() string
	AT() string
	Unmarshal(any) error
}

type Reply interface {
	Marshal() ([]byte, error)
}

var ErrOption = errors.New("invalid query option")

type Option struct {
	ID func() uuid.UUID
	RE func() []string
	AT func() string
}

func Options(from string, oo []any) (o Option, err error) {
	for _, v := range append([]any{from, uuid.New, []string(nil)}, oo...) {
		switch x := v.(type) {
		case func() uuid.UUID:
			o.ID = x
		case uuid.UUID:
			o.ID = func() uuid.UUID { return x }
		case func() []string:
			o.RE = x
		case []string:
			o.RE = func() []string { return x }
		case func() string:
			o.AT = x
		case string:
			o.AT = func() string { return x }
		case Query:
			o.ID = x.ID
			o.RE = x.RE
			o.AT = x.AT
		default:
			err = ErrOption
			return
		}
	}
	return
}
