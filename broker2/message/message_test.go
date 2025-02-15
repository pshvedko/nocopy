package message_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/stretchr/testify/require"

	"github.com/pshvedko/nocopy/broker2/message"
)

type Mediator []message.MiddlewareFunc

func (m Mediator) Middleware(string) []message.MiddlewareFunc {
	return m
}

func TestDecode(t *testing.T) {
	ctx := context.TODO()
	e := message.Envelope{
		ID:     uuid.New(),
		From:   "any",
		Return: []string{"path", "to", "home"},
		To:     "me",
		Type:   0,
		Method: "hello",
	}

	b, err := json.Marshal([]any{e, json.RawMessage{'{', '}'}})
	require.NoError(t, err)
	_, m, err := message.Decode(ctx, b, Mediator{})
	require.NoError(t, err)
	require.Equal(t, message.Raw{Envelope: e, Body: []byte{'{', '}'}}, m)

	b, err = json.Marshal([]any{})
	require.NoError(t, err)
	_, m, err = message.Decode(ctx, b, Mediator{})
	require.ErrorIs(t, err, message.ErrEmpty)
	require.Nil(t, m)

	b, err = json.Marshal([]any{e})
	require.NoError(t, err)
	_, m, err = message.Decode(ctx, b, Mediator{})
	require.ErrorIs(t, err, message.ErrNoPayload)
	require.Nil(t, m)

	b, err = json.Marshal([]any{e, json.RawMessage{'{', '}'}, json.RawMessage{'{', '}'}})
	require.NoError(t, err)
	_, m, err = message.Decode(ctx, b, Mediator{})
	require.ErrorIs(t, err, message.ErrRedundantMessage)
	require.Nil(t, m)

	b, err = json.Marshal([]any{e, json.RawMessage{'{', '"', 'a', '"', ':', '1', '}'}, json.RawMessage{'{', '}'}})
	require.NoError(t, err)
	ctx, m, err = message.Decode(ctx, b, Mediator{middleware})
	require.NoError(t, err)
	require.Equal(t, message.Raw{Envelope: e, Body: []byte{'{', '}'}}, m)
	require.Equal(t, ctx.Value(1), map[string]any{"a": 1.})
}

func middleware(ctx context.Context, bytes []byte) (context.Context, error) {
	var a interface{}
	err := json.Unmarshal(bytes, &a)
	if err != nil {
		return nil, err
	}
	return context.WithValue(ctx, 1, a), nil
}
