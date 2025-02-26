package log

import (
	"context"
	"fmt"
	"github.com/spf13/pflag"
	"io"
	"log/slog"
	"sync"
	"time"
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

type Handler struct {
	m sync.Locker
	w io.Writer
	l slog.Level
	a []slog.Attr
	g string
	h *Handler
}

func (h Handler) Enabled(_ context.Context, l slog.Level) bool {
	return h.l <= l
}

func (h Handler) Log(ctx context.Context) (group string, err error) {
	if h.h != nil {
		group, err = h.h.Log(ctx)
	}
	if h.g > "" {
		group += h.g
		group += "."
	}
	for _, v := range h.a {
		_, err = fmt.Fprintf(h.w, " %s%s=%s", group, v.Key, v.Value)
		if err != nil {
			return
		}
	}
	return
}

func (h Handler) Handle(ctx context.Context, r slog.Record) error {
	r.Attrs(func(a slog.Attr) bool {
		h.a = append(h.a, a)
		return true
	})
	h.m.Lock()
	defer h.m.Unlock()
	_, err := fmt.Fprintf(h.w, "%s [%s] %s",
		r.Time.Format(time.DateTime),
		r.Level,
		r.Message)
	if err != nil {
		return err
	}
	_, err = h.Log(ctx)
	return err
}

func (h Handler) WithAttrs(a []slog.Attr) slog.Handler {
	h.a = append(h.a, a...)
	return h
}

func (h Handler) WithGroup(group string) slog.Handler {
	return Handler{
		m: h.m,
		w: h.w,
		l: h.l,
		a: nil,
		g: group,
		h: &h,
	}
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
