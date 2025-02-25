package log

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/spf13/pflag"
)

type Level struct {
	p *slog.Level
}

func (l Level) String() string {
	return l.p.String()
}

func (l Level) Set(s string) error {
	return l.p.UnmarshalText([]byte(s))
}

func (l Level) Type() string {
	return "level"
}

func NewLogLevel(p *slog.Level, v slog.Level) pflag.Value {
	*p = v
	return Level{p: p}
}

type Attrs []slog.Attr

func (a Attrs) Format(f fmt.State, _ rune) {
	for _, v := range a {
		_, _ = fmt.Fprintf(f, " %s=%s", v.Key, v.Value)
	}
}

type Handler struct {
	m sync.Locker
	w io.Writer
	l slog.Level
	a Attrs
	g []string // FIXME
}

func (h Handler) Enabled(_ context.Context, l slog.Level) bool {
	return h.l <= l
}

func (h Handler) Handle(_ context.Context, r slog.Record) error {
	r.Attrs(func(a slog.Attr) bool {
		h.a = append(h.a, a)
		return true
	})
	h.m.Lock()
	defer h.m.Unlock()
	_, err := fmt.Fprintf(h.w, "%s [%s] %s%v\n",
		r.Time.Format(time.DateTime),
		r.Level,
		r.Message,
		h.a)
	return err
}

func (h Handler) WithAttrs(a []slog.Attr) slog.Handler {
	h.a = append(h.a, a...)
	return h
}

func (h Handler) WithGroup(g string) slog.Handler {
	h.g = append(h.g, g)
	return h
}

func NewHandler(w io.Writer, l slog.Level) Handler {
	return Handler{
		w: w,
		l: l,
		m: &sync.Mutex{},
	}
}

func NewLogger(w io.Writer, l slog.Level) *slog.Logger {
	return slog.New(NewHandler(w, l))
}
