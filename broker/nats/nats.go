package nats

import (
	"context"
	"net/url"

	"github.com/nats-io/nats.go"

	"github.com/pshvedko/nocopy/broker/message"
)

type Broker struct {
	*url.URL
	*nats.Conn
}

func (b Broker) Handle(method string, handler func(context.Context, message.Query) (message.Reply, error)) {
	//TODO implement me
	panic("implement me")
}

func (b Broker) Listen(ctx context.Context, at string) error {
	//TODO implement me
	panic("implement me")
}

func (b Broker) Send(ctx context.Context, to, by string, a any) error {
	//TODO implement me
	panic("implement me")
}

func (b Broker) Shutdown(ctx context.Context) error {
	//TODO implement me
	panic("implement me")
}

func New(ur1 *url.URL) (*Broker, error) {
	conn, err := nats.Connect(ur1.String())
	if err != nil {
		return nil, err
	}
	return &Broker{
		URL:  ur1,
		Conn: conn,
	}, nil
}
