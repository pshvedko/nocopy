package log

import (
	"log/slog"

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
