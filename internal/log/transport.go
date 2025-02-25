package log

import (
	"context"
	"log/slog"

	"github.com/pshvedko/nocopy/broker/exchange"
	"github.com/pshvedko/nocopy/broker/message"
)

type Transport struct {
	exchange.Transport
}

type Input struct {
	message.Decoder
}

func (i Input) Decode(ctx context.Context, bytes []byte) (context.Context, message.Message, error) {
	ctx, m, err := i.Decoder.Decode(ctx, bytes)
	if err != nil {
		slog.Error("READ", "error", err)
	}
	return ctx, m, err
}

func (i Input) Do(ctx context.Context, m message.Message) {
	slog.Debug("READ", "id", m.ID(), "by", m.Method(), "at", m.To(), "from", m.From(), "type", m.Type())
	i.Decoder.Do(ctx, m)
}

func (t Transport) Subscribe(ctx context.Context, at string, decoder message.Decoder) (exchange.Subscription, error) {
	slog.Debug("LISTEN", "at", at, "wide", true)
	return t.Transport.Subscribe(ctx, at, Input{Decoder: decoder})
}

func (t Transport) QueueSubscribe(ctx context.Context, at string, by string, decoder message.Decoder) (exchange.Subscription, error) {
	slog.Debug("LISTEN", "at", at)
	return t.Transport.QueueSubscribe(ctx, at, by, Input{Decoder: decoder})
}

func (t Transport) Publish(ctx context.Context, m message.Message, encoder message.Encoder) error {
	out := slog.With("id", m.ID(), "by", m.Method(), "at", m.From(), "to", m.To(), "type", m.Type())
	err := t.Transport.Publish(ctx, m, encoder)
	if err != nil {
		out = out.With("error", err)
	}
	out.Debug("SEND")
	return err
}
func (t Transport) Unsubscribe(topic exchange.Topic) error {
	switch topic.Wide() {
	case true:
		slog.Debug("FINISH", "at", topic, "wide", topic.Wide())
	default:
		slog.Debug("FINISH", "at", topic)
	}
	return t.Transport.Unsubscribe(topic)
}
