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
	WithReturn(...string) Builder
	WithError(error) Builder
	WithBody(any) Builder
	WithType(Type) Message // DEPRECATED:
	Reply() Builder
	Every() Builder
}

type Wrapper struct {
	Message
}

type WrapperEvery struct {
	Message
}

func (w WrapperEvery) Type() Type {
	return Broadcast
}

func (w Wrapper) Every() Builder {
	return Wrapper{Message: WrapperEvery{Message: w}}
}

type WrapperReply struct {
	Message
}

func (w WrapperReply) From() string {
	return w.Message.To()
}

func (w WrapperReply) To() string {
	return w.Message.From()
}

func (w WrapperReply) Type() Type {
	return w.Message.Type()&Failure | Answer
}

func (w Wrapper) Reply() Builder {
	return Wrapper{Message: WrapperReply{Message: w}}
}

type WrapperWithType struct {
	Message
	t Type
}

func (w WrapperWithType) Type() Type {
	return w.Type()
}

func (w Wrapper) WithType(t Type) Message {
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
	switch body := body.(type) {
	case nil:
		return w
	case Body:
		return Wrapper{Message: WrapperWithBody{Message: w, body: body}}
	default:
		return Wrapper{Message: WrapperWithBody{Message: w, body: NewBody(body)}}
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

type WrapperWithReturn struct {
	Message
	path []string
}

func (w WrapperWithReturn) Return() []string {
	return w.path
}

func (w Wrapper) WithReturn(path ...string) Builder {
	return Wrapper{Message: WrapperWithReturn{Message: w, path: path}}
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

func (w WrapperWithError) Error() Error {
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

func NewWithMessage(m Message) Builder {
	return Wrapper{
		Message: m,
	}
}

func New() Builder {
	return Wrapper{
		Message: Empty{id: uuid.New()},
	}
}
