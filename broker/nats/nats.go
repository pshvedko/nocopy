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

func (b Broker) Listen(ctx context.Context) error {
	//TODO implement me
	panic("implement me")
}

func (b Broker) Send(ctx context.Context, to string, a any) error {
	//TODO implement me
	panic("implement me")
}

func (b Broker) Shutdown(ctx context.Context) error {
	//TODO implement me
	panic("implement me")
}

func New(ur1 *url.URL) (*Broker, error) {
	nats.Options{
		Url:                         "",
		InProcessServer:             nil,
		Servers:                     nil,
		NoRandomize:                 false,
		NoEcho:                      false,
		Name:                        "",
		Verbose:                     false,
		Pedantic:                    false,
		Secure:                      false,
		TLSConfig:                   nil,
		TLSCertCB:                   nil,
		TLSHandshakeFirst:           false,
		RootCAsCB:                   nil,
		AllowReconnect:              false,
		MaxReconnect:                0,
		ReconnectWait:               0,
		CustomReconnectDelayCB:      nil,
		ReconnectJitter:             0,
		ReconnectJitterTLS:          0,
		Timeout:                     0,
		DrainTimeout:                0,
		FlusherTimeout:              0,
		PingInterval:                0,
		MaxPingsOut:                 0,
		ClosedCB:                    nil,
		DisconnectedCB:              nil,
		DisconnectedErrCB:           nil,
		ConnectedCB:                 nil,
		ReconnectedCB:               nil,
		DiscoveredServersCB:         nil,
		AsyncErrorCB:                nil,
		ReconnectBufSize:            0,
		SubChanLen:                  0,
		UserJWT:                     nil,
		Nkey:                        "",
		SignatureCB:                 nil,
		User:                        "",
		Password:                    "",
		Token:                       "",
		TokenHandler:                nil,
		Dialer:                      nil,
		CustomDialer:                nil,
		UseOldRequestStyle:          false,
		NoCallbacksAfterClientClose: false,
		LameDuckModeHandler:         nil,
		RetryOnFailedConnect:        false,
		Compression:                 false,
		ProxyPath:                   "",
		InboxPrefix:                 "",
		IgnoreAuthErrorAbort:        false,
		SkipHostLookup:              false,
	}.Connect()
	_, _ = nats.Connect(ur1.String())
	return &Broker{
		URL:  ur1,
		Conn: nil,
	}, nil
}
