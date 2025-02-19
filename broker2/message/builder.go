package message

import (
	"encoding/json"

	"github.com/google/uuid"
)

type Builder interface {
	Message
	WithID(uuid.UUID) Builder
	WithFrom(string) Builder
	WithTo(string) Builder
	WithMethod(string) Builder
	WithError(error) Builder
	WithBody(any) Builder
	Answer() Message
	Forward(string) Message
	Backward() Message
	Build() Message
}

type Wrapper struct {
	Message
}

func (w Wrapper) Build() Message {
	return w.Message
}

type WrapperBackward struct {
	Message
	n int
}

func (w WrapperBackward) From() string {
	return w.Message.To()
}

func (w WrapperBackward) Return() []string {
	return w.Message.Return()[:w.n]
}

func (w WrapperBackward) To() string {
	return w.Message.Return()[w.n]
}

func (w Wrapper) Backward() Message { return WrapperBackward{Message: w, n: len(w.Return()) - 1} }

type WrapperForward struct {
	Message
	to string
}

func (w WrapperForward) From() string {
	return w.Message.To()
}

func (w WrapperForward) Return() []string {
	return append(w.Message.Return(), w.Message.From())
}

func (w WrapperForward) To() string {
	return w.to
}

func (w WrapperForward) Type() Type {
	return Query
}

func (w Wrapper) Forward(to string) Message { return WrapperForward{Message: w, to: to} }

type WrapperAnswer struct {
	Message
}

func (w WrapperAnswer) From() string {
	return w.Message.To()
}

func (w WrapperAnswer) To() string {
	return w.Message.From()
}

func (w WrapperAnswer) Type() Type {
	return w.Message.Type()&Failure | Answer
}

func (w Wrapper) Answer() Message { return WrapperAnswer{Message: w} }

type WrapperWithType struct {
	Message
	t Type
}

func (w WrapperWithType) Type() Type {
	return w.t
}

func (w Wrapper) WithType(t Type) Builder {
	return Wrapper{Message: WrapperWithType{Message: w, t: t}}
}

type WrapperWithID struct {
	Message
	id uuid.UUID
}

func (w WrapperWithID) ID() uuid.UUID {
	return w.id
}

func (w Wrapper) WithID(id uuid.UUID) Builder {
	return Wrapper{Message: WrapperWithID{Message: w, id: id}}
}

type WrapperWithBody struct {
	Message
	body Body
}

func (w WrapperWithBody) Encode() ([]byte, error) {
	return w.body.Encode()
}

func (w WrapperWithBody) Decode(v any) error {
	b, err := w.body.Encode()
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

func (w Wrapper) WithBody(body any) Builder {
	switch b := body.(type) {
	case nil:
		return w
	case Body:
		return Wrapper{Message: WrapperWithBody{Message: w, body: b}}
	default:
		return Wrapper{Message: WrapperWithBody{Message: w, body: NewBody(b)}}
	}
}

type WrapperWithFrom struct {
	Message
	from string
}

func (w WrapperWithFrom) From() string {
	return w.from
}

func (w Wrapper) WithFrom(from string) Builder {
	return Wrapper{Message: WrapperWithFrom{Message: w, from: from}}
}

type WrapperWithTo struct {
	Message
	to string
}

func (w WrapperWithTo) To() string {
	return w.to
}

func (w Wrapper) WithTo(to string) Builder {
	return Wrapper{Message: WrapperWithTo{Message: w, to: to}}
}

type WrapperWithMethod struct {
	Message
	method string
}

func (w WrapperWithMethod) Method() string {
	return w.method
}

func (w Wrapper) WithMethod(method string) Builder {
	return Wrapper{Message: WrapperWithMethod{Message: w, method: method}}
}

type WrapperWithError struct {
	Message
	err Error
}

func (w WrapperWithError) Type() Type {
	return Failure
}

func (w WrapperWithError) Encode() ([]byte, error) {
	return w.err.Encode()
}

func (w WrapperWithError) Decode(any) error {
	return w.err
}

func (w Wrapper) WithError(err error) Builder {
	switch err := err.(type) {
	case nil:
		return w
	case Error:
		return Wrapper{Message: WrapperWithError{Message: w, err: err}}
	default:
		return Wrapper{Message: WrapperWithError{Message: w, err: NewError(500, err)}}
	}
}

func NewMessage(m Message) Builder {
	return Wrapper{Message: m}
}

type BuilderWithType interface {
	Builder
	WithType(Type) Builder
}

func NewWithID(id uuid.UUID) BuilderWithType {
	return Wrapper{Message: Empty{id: id}}
}

func New() BuilderWithType {
	return NewWithID(uuid.New())
}
