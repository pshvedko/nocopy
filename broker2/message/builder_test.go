package message

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWrapper_Build(t *testing.T) {
	m := New().
		WithType(Synchro).
		WithTo("100").
		WithFrom("000").
		Forward("200")

	require.Equal(t, "200", m.To())
	require.Equal(t, "100", m.From())
	require.Equal(t, []string{"000"}, m.Return())
	require.Equal(t, Query, m.Type())

	m = NewMessage(m).
		WithFrom("111").
		Build()

	require.Equal(t, "200", m.To())
	require.Equal(t, "111", m.From())
	require.Equal(t, []string{"000"}, m.Return())
	require.Equal(t, Query, m.Type())

	m = NewMessage(m).
		Forward("300")

	require.Equal(t, "300", m.To())
	require.Equal(t, "200", m.From())
	require.Equal(t, []string{"000", "111"}, m.Return())
	require.Equal(t, Query, m.Type())

	m = NewMessage(m).
		Answer()

	require.Equal(t, "200", m.To())
	require.Equal(t, "300", m.From())
	require.Equal(t, []string{"000", "111"}, m.Return())
	require.Equal(t, Answer, m.Type())

	m = NewMessage(m).
		Backward()

	require.Equal(t, "111", m.To())
	require.Equal(t, "200", m.From())
	require.Equal(t, []string{"000"}, m.Return())
	require.Equal(t, Answer, m.Type())

	m = NewMessage(m).
		Backward()

	require.Equal(t, "000", m.To())
	require.Equal(t, "111", m.From())
	require.Equal(t, []string{}, m.Return())
	require.Equal(t, Answer, m.Type())
}
