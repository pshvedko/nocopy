package message

import (
	"encoding/json"
	"errors"

	"github.com/google/uuid"
)

type Address interface {
	ID() uuid.UUID
	RE() []string
	AT() string
}

type Header interface {
	Address
	FF() [3]string
	BY() string
	TO() string
}

type Query interface {
	Header
	Unmarshal(any) error
}

type Reply interface {
	Marshal() ([]byte, error)
}

var ErrOption = errors.New("invalid query option")

type Option struct {
	id func() uuid.UUID
	re func() []string
	at func() string
}

func (o Option) ID() uuid.UUID {
	return o.id()
}

func (o Option) RE() []string {
	return o.re()
}

func (o Option) AT() string {
	return o.at()
}

func MakeAddress(at string, oo []any) (Address, error) {
	var o Option
	var err error
	for _, v := range append([]any{at, uuid.New, []string(nil)}, oo...) {
		switch x := v.(type) {
		case func() uuid.UUID:
			o.id = x
		case uuid.UUID:
			o.id = func() uuid.UUID { return x }
		case func() []string:
			o.re = x
		case []string:
			o.re = func() []string { return x }
		case func() string:
			o.at = x
		case string:
			o.at = func() string { return x }
		case Address:
			o.id = x.ID
			o.re = x.RE
			o.at = x.AT
		default:
			return nil, ErrOption
		}
	}
	return o, err
}

type Body struct {
	Any any
}

func (b Body) Marshal() ([]byte, error) {
	if b.Any != nil {
		return json.Marshal(b.Any)
	}
	return nil, nil
}

type Toward struct {
	To, By string
	Address
}

func (h Toward) FF() [3]string {
	return [3]string{"F"}
}

func (h Toward) BY() string {
	return h.By
}

func (h Toward) TO() string {
	return h.To
}

type Backward struct {
	Header
}

func (b Backward) FF() [3]string {
	ff := b.Header.FF()
	ff[0] = "R"
	return ff
}
