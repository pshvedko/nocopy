package message

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/stretchr/testify/require"
)

func TestDecode(t *testing.T) {
	ctx := context.TODO()
	m := Envelope{
		ID:     uuid.New(),
		From:   "any",
		Return: []string{"path", "to", "home"},
		To:     "me",
		Type:   2,
		Handle: "hello",
	}

	b, err := json.Marshal([]any{m, json.RawMessage{'{', '}'}})
	require.NoError(t, err)
	_, message, err := Decode(ctx, t.Name(), b)
	require.NoError(t, err)
	require.Equal(t, Raw{Envelope: m, Payload: []byte{'{', '}'}}, message)

	b, err = json.Marshal([]any{})
	require.NoError(t, err)
	_, message, err = Decode(ctx, t.Name(), b)
	require.ErrorIs(t, err, ErrEmpty)
	require.Nil(t, message)

	b, err = json.Marshal([]any{m})
	require.NoError(t, err)
	_, message, err = Decode(ctx, t.Name(), b)
	require.ErrorIs(t, err, ErrNoPayload)
	require.Nil(t, message)

	b, err = json.Marshal([]any{m, json.RawMessage{'{', '}'}, json.RawMessage{'{', '}'}})
	require.NoError(t, err)
	_, message, err = Decode(ctx, t.Name(), b)
	require.ErrorIs(t, err, ErrRedundantMessage)
	require.Nil(t, message)

	b, err = json.Marshal([]any{m, json.RawMessage{'{', '"', 'a', '"', ':', '1', '}'}, json.RawMessage{'{', '}'}})
	require.NoError(t, err)
	ctx, message, err = Decode(ctx, t.Name(), b, middleware())
	require.NoError(t, err)
	require.Equal(t, Raw{Envelope: m, Payload: []byte{'{', '}'}}, message)
	require.Equal(t, ctx.Value(1), map[string]any{"a": 1.})
}

func middleware() MiddlewareFunc {
	return func(ctx context.Context, bytes []byte) (context.Context, error) {
		var a interface{}
		err := json.Unmarshal(bytes, &a)
		if err != nil {
			return nil, err
		}
		return context.WithValue(ctx, 1, a), nil
	}
}
