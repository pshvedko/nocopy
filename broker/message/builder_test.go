package message

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestWrapper_Build(t *testing.T) {
	id := uuid.New()
	m := New().
		WithType(Request).
		WithID(id).
		WithTo("100").
		WithFrom("000").
		Build()

	require.Equal(t, "100", m.To())
	require.Equal(t, "000", m.From())
	require.Equal(t, []string{}, m.Return())
	require.Equal(t, Request, m.Type())
	require.Equal(t, id, m.ID())

	m = NewMessage(m).
		Forward("200")

	require.Equal(t, "200", m.To())
	require.Equal(t, "100", m.From())
	require.Equal(t, []string{"000"}, m.Return())
	require.Equal(t, Query, m.Type())
	require.Equal(t, id, m.ID())

	m = NewMessage(m).
		WithFrom("111").
		Build()

	require.Equal(t, "200", m.To())
	require.Equal(t, "111", m.From())
	require.Equal(t, []string{"000"}, m.Return())
	require.Equal(t, Query, m.Type())
	require.Equal(t, id, m.ID())

	m = NewMessage(m).
		Forward("300")

	require.Equal(t, "300", m.To())
	require.Equal(t, "200", m.From())
	require.Equal(t, []string{"000", "111"}, m.Return())
	require.Equal(t, Query, m.Type())
	require.Equal(t, id, m.ID())

	m = NewMessage(m).
		Answer()

	require.Equal(t, "200", m.To())
	require.Equal(t, "300", m.From())
	require.Equal(t, []string{"000", "111"}, m.Return())
	require.Equal(t, Answer, m.Type())
	require.Equal(t, id, m.ID())

	m = NewMessage(m).
		Backward()

	require.Equal(t, "111", m.To())
	require.Equal(t, "200", m.From())
	require.Equal(t, []string{"000"}, m.Return())
	require.Equal(t, Answer, m.Type())
	require.Equal(t, id, m.ID())

	m = NewMessage(m).
		WithError(NewError(123, ErrEmpty)).
		Backward()

	require.Equal(t, "000", m.To())
	require.Equal(t, "111", m.From())
	require.Equal(t, []string{}, m.Return())
	require.Equal(t, Failure, m.Type())
	require.Equal(t, id, m.ID())
	require.ErrorIs(t, Error{123, "empty"}, m.Decode(struct{}{}))
}

func TestWrapperAnswer_Type(t *testing.T) {
	tests := []struct {
		name    string
		message Message
		want    Type
	}{
		// TODO: Add test cases.
		{
			name:    "",
			message: New().WithType(Query).Answer(),
			want:    Answer,
		},
		{
			name:    "",
			message: New().WithType(Answer).Answer(),
			want:    Answer,
		},
		{
			name:    "",
			message: New().WithType(Request).Answer(),
			want:    Answer,
		},
		{
			name:    "",
			message: New().WithType(Failure).Answer(),
			want:    Failure,
		},
		{
			name:    "",
			message: New().WithType(Broadcast).Answer(),
			want:    Answer,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.message.Type(); got != tt.want {
				t.Errorf("Type() = %v, want %v", got, tt.want)
			}
		})
	}
}
