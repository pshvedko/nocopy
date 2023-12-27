package message

import (
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
	OF() [3]string
	BY() string
	TO() string
}

type Message interface {
	Header
	Reply
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

type Toward struct {
	To, By string
	Address
}

func (h Toward) OF() [3]string {
	return [3]string{"F"}
}

func (h Toward) BY() string {
	return h.By
}

func (h Toward) TO() string {
	return h.To
}

type Respond struct {
	Header
}

func (h Respond) OF() [3]string {
	ff := h.Header.OF()
	ff[0] = "R"
	return ff
}

type Forward struct {
	To string
	Message
}

func (h Forward) TO() string {
	return h.To
}

func (h Forward) RE() []string {
	return append(h.Message.RE(), h.Message.TO())
}

type Backward struct {
	Message
}

func (h Backward) TO() string {
	re := h.Message.RE()
	if n := len(re); n > 0 {
		return re[n-1]
	}
	return ""
}

func (h Backward) RE() []string {
	re := h.Message.RE()
	if n := len(re); n > 0 {
		return re[:n-1]
	}
	return nil
}
