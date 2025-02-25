package service

import (
	"context"
	"encoding/json"
)

type Auth struct{}

func (a Auth) Decode(ctx context.Context, bytes []byte) (context.Context, error) {
	return ctx, json.Unmarshal(bytes, &map[string]any{})
}

func (a Auth) Encode(context.Context) ([]byte, error) {
	return json.Marshal(map[string]any{})
}
