package message

import (
	"encoding/json"
	"errors"
	"reflect"

	"github.com/google/uuid"
)

type Builder interface {
	Message
	WithMessage(Message) Builder
	WithID(uuid.UUID) Builder
	WithError(error) Builder
	WithBody(any) Builder
	WithFrom(string) Builder
	WithTo(string) Builder
	WithInvert() Builder
	WithMethod(string) Builder
	WithPath(...string) Builder
	WithStraight(bool) Builder
}

type Wrapper struct {
	Message
}

type Straight struct {
	Message
	of bool
}

func (m Straight) OF() [3]string {
	of := m.Message.OF()
	if m.of {
		of[0] = "F"
	} else {
		of[0] = "R"
	}
	return of
}

func (m Wrapper) WithStraight(of bool) Builder {
	return Wrapper{Message: Straight{Message: m, of: of}}
}

func (m Wrapper) WithMessage(copy Message) Builder { return Wrapper{Message: copy} }

type Path struct {
	Message
	re []string
}

func (m Path) RE() []string { return m.re }

func (m Wrapper) WithPath(re ...string) Builder { return Wrapper{Message: Path{Message: m, re: re}} }

type Method struct {
	Message
	by string
}

func (m Method) BY() string { return m.by }

func (m Wrapper) WithMethod(by string) Builder { return Wrapper{Message: Method{Message: m, by: by}} }

type Id struct {
	Message
	id uuid.UUID
}

func (m Wrapper) WithID(id uuid.UUID) Builder { return Wrapper{Message: Id{Message: m, id: id}} }

type Invert struct {
	Message
}

func (m Invert) OF() [3]string {
	of := m.Message.OF()
	if of[0] == "R" {
		of[0] = "F"
	} else {
		of[0] = "R"
	}
	return of
}

func (m Wrapper) WithInvert() Builder { return Wrapper{Message: Invert{Message: m}} }

type To struct {
	Message
	to string
}

func (m To) TO() string { return m.to }

func (m Wrapper) WithTo(to string) Builder { return Wrapper{Message: To{Message: m, to: to}} }

type From struct {
	Message
	from string
}

func (m From) AT() string { return m.from }

func (m Wrapper) WithFrom(from string) Builder { return Wrapper{Message: From{Message: m, from: from}} }

type Error struct {
	Message
	err string
}

func (m Error) OF() [3]string {
	of := m.Message.OF()
	of[1] = m.err
	return of
}

func (m Wrapper) WithError(err error) Builder {
	return Wrapper{Message: Error{Message: m, err: err.Error()}}
}

type Body struct {
	Message
	body any
}

func (m Body) Unmarshal(to any) error {
	of := m.OF()
	if len(of[1]) > 0 {
		return errors.New(of[1])
	}
	reflect.ValueOf(to).Elem().Set(reflect.ValueOf(m.body)) // TODO
	return nil
}

func (m Body) Marshal() ([]byte, error) { return json.Marshal(m.body) }

func (m Wrapper) WithBody(body any) Builder { return Wrapper{Message: Body{Message: m, body: body}} }

type Empty struct{}

func (m Empty) ID() uuid.UUID { return uuid.UUID{} }

func (m Empty) RE() []string { return nil }

func (m Empty) AT() string { return "" }

func (m Empty) OF() [3]string { return [3]string{"F"} }

func (m Empty) BY() string { return "" }

func (m Empty) TO() string { return "" }

func (m Empty) Marshal() ([]byte, error) { return nil, nil }

func (m Empty) Unmarshal(any) error { return nil }

func New() Builder {
	return Wrapper{Message: Empty{}}
}
